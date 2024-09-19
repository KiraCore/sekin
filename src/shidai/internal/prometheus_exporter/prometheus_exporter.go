package main

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// static value
var (
	// Total number of CPU cores
	totalCPUCores = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_total_cores",
			Help: "Total number of CPU cores available.",
		},
	)

	// Total amount of RAM
	totalRAM = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ram_total",
			Help: "Total amount of RAM available (in bytes).",
		},
	)

	// Total disk space
	totalDiskSpace = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "disk_total",
			Help: "Total disk space available (in bytes).",
		},
	)

	// Total bandwidth
	totalBandwidth = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "bandwidth_total",
			Help: "Total available bandwidth (in bits per second).",
		},
	)

	// Total GPU CUDA cores
	totalGPUCUDACores = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gpu_total_cuda_cores",
			Help: "Total number of GPU CUDA cores available.",
		},
	)

	// Total VRAM
	totalVRAM = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gpu_total_vram",
			Help: "Total VRAM available (in bytes).",
		},
	)

	// Total CPU GHz
	totalCPUGHz = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_total_ghz",
			Help: "Total CPU GHz available (sum of maximum frequencies of all cores).",
		},
	)
)

// dynamic values
var (
	currentCPUGHz = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_current_ghz",
			Help: "Current CPU GHz in use (sum of current frequencies of all cores).",
		},
	)
)

// run this in anonym func
func RunPrometheusExporterService(ctx context.Context) error {
	registerMetrics()

	err := staticValueGetter()
	if err != nil {
		return nil
	}

	updatePeriod := time.Second * 4

	ticker := time.NewTicker(updatePeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err = dynamicValueGetter()
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

func registerMetrics() {
	prometheus.MustRegister(totalCPUCores)
	prometheus.MustRegister(totalRAM)
	prometheus.MustRegister(totalDiskSpace)
	prometheus.MustRegister(totalBandwidth)
	prometheus.MustRegister(totalGPUCUDACores)
	prometheus.MustRegister(totalVRAM)
	prometheus.MustRegister(totalCPUGHz)
	prometheus.MustRegister(currentCPUGHz)
}

func staticValueGetter() error {
	return nil
}
func dynamicValueGetter() error {
	return nil
}
