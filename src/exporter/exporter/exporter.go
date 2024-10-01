package exporter

import (
	"context"
	"time"

	"github.com/kiracore/sekin/src/exporter/logger"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var log = logger.GetLogger()

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

	uploadBandwidth = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "bandwidth_upload",
			Help: "Upload bandwidth (in bits per second).",
		},
	)
	downloadBandwidth = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "bandwidth_download",
			Help: "Download bandwidth (in bits per second).",
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

// run this in anonym func
func RunPrometheusExporterService(ctx context.Context) {
	staticValueUpdater()
	updatePeriod := time.Second * 4
	ticker := time.NewTicker(updatePeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			dynamicValueGetter()

		case <-ctx.Done():
			return
		}
	}
}

func RegisterMetrics() *prometheus.Registry {
	var customRegistry = prometheus.NewRegistry()
	customRegistry.MustRegister(
		totalCPUCores,
		totalRAM,
		totalDiskSpace,
		uploadBandwidth,
		downloadBandwidth,
		totalVRAM,
		totalCPUGHz,
		totalGPUCUDACores,
	)
	return customRegistry
}

func staticValueUpdater() {
	if err := collectTotalCPUCores(); err != nil {
		log.Warn("unable to collect total value of cpu cores", zap.Error(err))
	}
	if err := collectTotalBandwidth(); err != nil {
		log.Warn("unable to collect total value of cpu cores", zap.Error(err))
	}
	if err := collectTotalCPUGHz(); err != nil {
		log.Warn("unable to collect total value of cpu cores", zap.Error(err))
	}
	if err := collectTotalVRAM(); err != nil {
		log.Warn("unable to collect total value of cpu cores", zap.Error(err))
	}
	if err := collectTotalRAM(); err != nil {
		log.Warn("unable to collect total value of cpu cores", zap.Error(err))
	}
	if err := collectTotalGPUCUDACores(); err != nil {
		log.Warn("unable to collect total value of cpu cores", zap.Error(err))
	}
	if err := collectTotalDiskSpace(); err != nil {
		log.Warn("unable to collect total value of cpu cores", zap.Error(err))
	}
}
func dynamicValueGetter() {

}
