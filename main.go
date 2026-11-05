package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/libvirt/libvirt-go"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type App struct {
	mu   sync.RWMutex
	vms  []string
	conn *libvirt.Connect
}

func (a *App) Load_vms() error {
	file, err := os.ReadFile("vms")
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.vms = strings.Split(string(file), "\n")
	a.mu.Unlock()
	return nil
}

func (a *App) Save_vms() error {
	file, err := os.Create("vms")
	if err != nil {
		return err
	}
	defer file.Close()

	a.mu.RLock()
	vms := append([]string(nil), a.vms...)
	a.mu.RUnlock()

	for _, i := range vms {
		if i == "" {
			continue
		}
		_, err := a.conn.LookupDomainByName(i)
		if err != nil {
			log.Printf("%v", err)
		}

		fmt.Fprintf(file, "%s\n", i)
	}
	return nil
}

func (a *App) save_vms(w http.ResponseWriter, r *http.Request) {
	if err := a.Save_vms(); err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		log.Printf("%v", err)
		return
	}
	fmt.Fprintf(w, "Succsessfully saved")
	log.Printf("%v", "Succsessfully saved")
}

func (a *App) suspend_vm(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	dom, err := a.conn.LookupDomainByName(name)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		log.Printf("%v", err)
	}
	dom.Suspend()
	fmt.Fprintf(w, "%v", "Succsess")
	log.Printf("Succsess")
}

func (a *App) resume_vm(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	dom, err := a.conn.LookupDomainByName(name)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		log.Printf("%v", err)
		return
	}
	dom.Resume()
	fmt.Fprintf(w, "%v", "Succsess")
	log.Printf("Succsess")
}

func (a *App) kill_vm(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	dom, err := a.conn.LookupDomainByName(name)
	if err != nil {
		http.Error(w, "No such vm", http.StatusBadRequest)
		log.Printf("No such vm")
		return
	}
	dom.Destroy()
	dom.Undefine()

	a.mu.Lock()
	for i, v := range a.vms {
		if v == name {
			a.vms = append(a.vms[:i], a.vms[i+1:]...)
			break
		}
	}
	a.mu.Unlock()
}

func (a *App) autosave() {
	for {
		a.Save_vms()
		time.Sleep(3 * time.Second)
	}
}

func (a *App) Start() {
	con, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		log.Fatal("Error with qemu")
	}

	a.conn = con

	defer a.conn.Close()

	err = a.Load_vms()
	if err != nil {
		log.Fatal("Something went wrong wile loading list of previous vm's", err)
	}

	for _, i := range a.vms {
		if i == "" {
			continue
		}
		dom, err := a.conn.LookupDomainByName(i)
		if err != nil {
			log.Printf("%v", err)
		} else {
			dom.Resume()
		}
	}

	go a.autosave()

	http.HandleFunc("/", a.index)
	http.HandleFunc("/api/vms", a.listVMs)
	http.HandleFunc("/create", a.create_vm)
	http.HandleFunc("/save", a.save_vms)
	http.HandleFunc("/suspend_vm", a.suspend_vm)
	http.HandleFunc("/resume_vm", a.resume_vm)
	http.HandleFunc("/kill_vm", a.kill_vm)
	http.Handle("/metrics", a.metricsHandler())

	http.ListenAndServe(":8080", nil)
}

func (a *App) create_vm(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	memory, err := strconv.ParseInt(r.URL.Query().Get("memory"), 10, 32)
	if err != nil {
		log.Printf("Incorrect memory format, setting up to 512MiB")
		fmt.Fprintf(w, "Incorrect memory format, setting up to 512MiB")
		memory = 512
	}
	cpus, err := strconv.ParseInt(r.URL.Query().Get("cpu"), 10, 32)
	if err != nil {
		log.Printf("Incorrect cpu format, setting up to 1")
		fmt.Fprintf(w, "Incorrect cpu format, setting up to 1")
		cpus = 1
	}

	cpu_count, err := cpu.Counts(true)
	if err != nil {
		log.Printf("Error counting machine availiable resources")
		http.Error(w, "Error counting machine availiable resources", http.StatusBadRequest)
		return
	}
	cmem, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error counting machine availiable resources")
		http.Error(w, "Error counting machine availiable resources", http.StatusBadRequest)
		return
	}
	if int64(cpu_count) < cpus || int64(cmem.Total/1024/1024) < memory {
		log.Printf("Not enough resources for creating vm")
		http.Error(w, "Not enough resources for creating vm", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("cloud-init.iso")
	if err != nil {
		http.Error(w, "Ошибка получения файла: "+err.Error(), http.StatusBadRequest)
		log.Printf("Ошибка получения файла: %v", err)
		return
	}
	defer file.Close()

	filename := fmt.Sprintf("cloud-init-%s.iso", uuid.New().String())
	dstPath := "/app/tmp/" + filename
	log.Printf(dstPath)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Ошибка создания файла: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Ошибка создания файла: " + err.Error())
		return
	}

	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Ошибка сохранения файла: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Ошибка сохранения файла: " + err.Error())
		return
	}

	imageDir := os.Getenv("CLOUDMAKER_IMAGE_DIR")

	xmlConfig := fmt.Sprintf(`
	<domain type='kvm'>
	  <name>%v</name>
	  <memory unit='MiB'>%v</memory>
	  <vcpu>%v</vcpu>
	  <os>
	    <type arch='x86_64'>hvm</type>
	  </os>
	  <devices>
	    <emulator>/usr/bin/qemu-system-x86_64</emulator>
		<disk type='file' device='cdrom'>
      		<driver name='qemu' type='raw'/>
      		<source file='%v'/>
      		<target dev='hda' bus='ide'/>
      		<readonly/>
    	</disk>
	  </devices>
	</domain>`, name, memory, cpus, imageDir+"/"+filename)

	_, err = a.conn.DomainCreateXML(xmlConfig, 0)
	if err != nil {
		http.Error(w, "Error creating virtual machine", http.StatusInternalServerError)
		log.Printf("Error creating virtual machine %v, %v", name, err)
		return
	}

	a.mu.Lock()
	a.vms = append(a.vms, name)
	a.mu.Unlock()

	fmt.Fprintf(w, "Виртуальная машина '%s' успешно запущена.\n", name)
	log.Printf("Виртуальная машина '%s' успешно запущена.\n", name)
}

func main() {
	ap := App{}
	ap.Start()
}
