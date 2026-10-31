package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

var (
	cpuTotalDesc = prometheus.NewDesc(
		"cloudmaker_cpu_total",
		"Number of logical CPU cores available on the host.",
		nil, nil,
	)
	memoryTotalDesc = prometheus.NewDesc(
		"cloudmaker_memory_total_bytes",
		"Total host memory in bytes.",
		nil, nil,
	)
	memoryUsedDesc = prometheus.NewDesc(
		"cloudmaker_memory_used_bytes",
		"Used host memory in bytes.",
		nil, nil,
	)
	memoryAvailableDesc = prometheus.NewDesc(
		"cloudmaker_memory_available_bytes",
		"Available host memory in bytes.",
		nil, nil,
	)
	diskTotalDesc = prometheus.NewDesc(
		"cloudmaker_disk_total_bytes",
		"Total size of the filesystem holding / in bytes.",
		[]string{"path"}, nil,
	)
	diskUsedDesc = prometheus.NewDesc(
		"cloudmaker_disk_used_bytes",
		"Used space of the filesystem holding / in bytes.",
		[]string{"path"}, nil,
	)
	diskFreeDesc = prometheus.NewDesc(
		"cloudmaker_disk_free_bytes",
		"Free space of the filesystem holding / in bytes.",
		[]string{"path"}, nil,
	)
	vmsTotalDesc = prometheus.NewDesc(
		"cloudmaker_vms_total",
		"Number of virtual machines tracked by CloudMaker.",
		nil, nil,
	)
	vmCPUDesc = prometheus.NewDesc(
		"cloudmaker_vm_cpu",
		"Number of virtual CPUs assigned to a virtual machine.",
		[]string{"name"}, nil,
	)
	vmMemoryDesc = prometheus.NewDesc(
		"cloudmaker_vm_memory_limit_bytes",
		"Memory limit assigned to a virtual machine in bytes.",
		[]string{"name"}, nil,
	)
	vmStateDesc = prometheus.NewDesc(
		"cloudmaker_vm_state",
		"Virtual machine state (0 nostate, 1 running, 2 blocked, 3 paused, 4 shutdown, 5 shutoff, 6 crashed, 7 pmsuspended).",
		[]string{"name"}, nil,
	)
)

type metricsCollector struct {
	app *App
}

func (c metricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cpuTotalDesc
	ch <- memoryTotalDesc
	ch <- memoryUsedDesc
	ch <- memoryAvailableDesc
	ch <- diskTotalDesc
	ch <- diskUsedDesc
	ch <- diskFreeDesc
	ch <- vmsTotalDesc
	ch <- vmCPUDesc
	ch <- vmMemoryDesc
	ch <- vmStateDesc
}

func (c metricsCollector) Collect(ch chan<- prometheus.Metric) {
	if count, err := cpu.Counts(true); err == nil {
		ch <- prometheus.MustNewConstMetric(cpuTotalDesc, prometheus.GaugeValue, float64(count))
	}

	if m, err := mem.VirtualMemory(); err == nil {
		ch <- prometheus.MustNewConstMetric(memoryTotalDesc, prometheus.GaugeValue, float64(m.Total))
		ch <- prometheus.MustNewConstMetric(memoryUsedDesc, prometheus.GaugeValue, float64(m.Used))
		ch <- prometheus.MustNewConstMetric(memoryAvailableDesc, prometheus.GaugeValue, float64(m.Available))
	}

	if d, err := disk.Usage("/"); err == nil {
		ch <- prometheus.MustNewConstMetric(diskTotalDesc, prometheus.GaugeValue, float64(d.Total), "/")
		ch <- prometheus.MustNewConstMetric(diskUsedDesc, prometheus.GaugeValue, float64(d.Used), "/")
		ch <- prometheus.MustNewConstMetric(diskFreeDesc, prometheus.GaugeValue, float64(d.Free), "/")
	}

	vms := 0
	if c.app.conn != nil {
		c.app.mu.RLock()
		names := append([]string(nil), c.app.vms...)
		c.app.mu.RUnlock()

		for _, name := range names {
			if name == "" {
				continue
			}
			dom, err := c.app.conn.LookupDomainByName(name)
			if err != nil {
				continue
			}
			info, err := dom.GetInfo()
			dom.Free()
			if err != nil {
				continue
			}

			vms++
			ch <- prometheus.MustNewConstMetric(vmCPUDesc, prometheus.GaugeValue, float64(info.NrVirtCpu), name)
			ch <- prometheus.MustNewConstMetric(vmMemoryDesc, prometheus.GaugeValue, float64(info.MaxMem)*1024, name)
			ch <- prometheus.MustNewConstMetric(vmStateDesc, prometheus.GaugeValue, float64(info.State), name)
		}
	}
	ch <- prometheus.MustNewConstMetric(vmsTotalDesc, prometheus.GaugeValue, float64(vms))
}

func (a *App) metricsHandler() http.Handler {
	registry := prometheus.NewRegistry()
	registry.MustRegister(metricsCollector{app: a})
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}
