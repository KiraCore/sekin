package exporter

import (
	"context"
	"fmt"
	"time"

	"github.com/jaypipes/ghw/pkg/gpu"
	"github.com/kiracore/sekin/src/exporter/logger"
	systeminfo "github.com/kiracore/sekin/src/exporter/system_info"
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
		totalCPUGHz,
	)
	err := gatherGpusGauges(customRegistry)
	if err != nil {
		log.Debug("Unable to register gpu gauges", zap.Error(err))
	}
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

	if err := collectTotalRAM(); err != nil {
		log.Warn("unable to collect total value of cpu cores", zap.Error(err))
	}

	if err := collectTotalDiskSpace(); err != nil {
		log.Warn("unable to collect total value of cpu cores", zap.Error(err))
	}
}
func dynamicValueGetter() {

}

// adds to registry all graphics card if available
func gatherGpusGauges(reg *prometheus.Registry) error {
	gpus, err := systeminfo.CollectGpusInfo()
	if err != nil {
		return err
	}
	gpus_gauges := []*prometheus.GaugeVec{}
	for i, gpu := range gpus {
		gauge, err := create_gpu_gauge(i, gpu)
		if err != nil {
			log.Debug("error getting gauge values", zap.String("gpu address", gpu.Address), zap.Error(err))
			continue
		}
		gpus_gauges = append(gpus_gauges, gauge)
	}
	for _, g := range gpus_gauges {
		err := reg.Register(g)
		if err != nil {
			log.Debug("unable to register metric", zap.Any("gauge", g), zap.Error(err))
		}
	}
	return nil
}

func create_gpu_gauge(gpuNum int, gpuInfo *gpu.GraphicsCard) (*prometheus.GaugeVec, error) {
	vendor := gpuInfo.DeviceInfo.Vendor.ID

	//for more info about vendor id use https://pci-ids.ucw.cz/
	switch vendor {

	case "1002": // amd vendor id
		return create_amd_gpu_gauge(gpuNum, gpuInfo)
	case "10DE": //nvidia vendor id
		return create_nvidia_gpu_gauge(gpuNum, gpuInfo)
	case "8086": // should be a intel controller, need to double check
		return create_intel_gpu_gauge(gpuNum, gpuInfo)
	default:
		return nil, fmt.Errorf("unable to detect GPU device, device info: %+v, vendor ID: %s", gpuInfo.DeviceInfo, vendor)
	}
}
func create_amd_gpu_gauge(gpuNum int, gpuInfo *gpu.GraphicsCard) (*prometheus.GaugeVec, error) {
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: fmt.Sprintf("gpu_%v", gpuNum),
		Help: fmt.Sprintf("Device info for %v Model id: \"%v\"", gpuInfo.DeviceInfo.Product.Name, gpuInfo.DeviceInfo.Product.ID),
	}, []string{"property"})

	err := prometheus.Register(gauge)
	if err != nil {
		return nil, fmt.Errorf("error registering gpu gauge: %v", err)
	}

	vram, err := get_amd_gpu_vram(gpuInfo)
	if err != nil {
		return nil, fmt.Errorf("error getting GPU VRAM: %v", err)
	}
	gauge.With(prometheus.Labels{"property": "vram"}).Set(float64(vram))

	return gauge, nil
}
func create_nvidia_gpu_gauge(gpuNum int, gpuInfo *gpu.GraphicsCard) (*prometheus.GaugeVec, error) {
	return nil, nil
}
func create_intel_gpu_gauge(gpuNum int, gpuInfo *gpu.GraphicsCard) (*prometheus.GaugeVec, error) {
	return nil, nil
}
