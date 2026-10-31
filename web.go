package main

import (
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/libvirt/libvirt-go"
)

//go:embed web/index.html
var indexHTML []byte

type vmView struct {
	Name        string `json:"name"`
	State       string `json:"state"`
	StateCode   int    `json:"state_code"`
	CPUs        uint   `json:"cpus"`
	MemoryBytes uint64 `json:"memory_bytes"`
}

func domainStateName(state libvirt.DomainState) string {
	switch state {
	case libvirt.DOMAIN_RUNNING:
		return "running"
	case libvirt.DOMAIN_BLOCKED:
		return "blocked"
	case libvirt.DOMAIN_PAUSED:
		return "paused"
	case libvirt.DOMAIN_SHUTDOWN:
		return "shutdown"
	case libvirt.DOMAIN_SHUTOFF:
		return "shutoff"
	case libvirt.DOMAIN_CRASHED:
		return "crashed"
	case libvirt.DOMAIN_PMSUSPENDED:
		return "pmsuspended"
	default:
		return "nostate"
	}
}

func (a *App) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (a *App) listVMs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.mu.RLock()
	names := append([]string(nil), a.vms...)
	a.mu.RUnlock()

	vms := make([]vmView, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}

		vm := vmView{Name: name, State: "not found", StateCode: -1}
		if a.conn != nil {
			dom, err := a.conn.LookupDomainByName(name)
			if err == nil {
				info, infoErr := dom.GetInfo()
				dom.Free()
				if infoErr == nil {
					vm.State = domainStateName(info.State)
					vm.StateCode = int(info.State)
					vm.CPUs = info.NrVirtCpu
					vm.MemoryBytes = info.MaxMem * 1024
				}
			}
		}
		vms = append(vms, vm)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vms)
}
