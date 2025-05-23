package exporter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jaypipes/ghw/pkg/gpu"
	"github.com/kiracore/sekin/src/exporter/logger"
	systeminfo "github.com/kiracore/sekin/src/exporter/system_info"
	"github.com/kiracore/sekin/src/exporter/types"
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
	if gpuInfo.DeviceInfo != nil {
		vendor := gpuInfo.DeviceInfo.Vendor.ID
		//for more info about vendor id use https://pci-ids.ucw.cz/
		switch strings.ToLower(vendor) {
		case strings.ToLower(types.VENDOR_AMD_GPU_ID): // amd vendor id
			return create_amd_gpu_gauge(gpuNum, gpuInfo)
		case strings.ToLower(types.VENDOR_NVIDIA_GPU_ID): //nvidia vendor id
			return create_nvidia_gpu_gauge(gpuNum, gpuInfo)
		case strings.ToLower(types.VENDOR_INTEL_GPU_ID): // should be a intel controller, need to double check
			return create_intel_gpu_gauge(gpuNum, gpuInfo)
		default:
			return nil, fmt.Errorf("unable to detect GPU device, device info: %+v, vendor ID: %s", gpuInfo.DeviceInfo, vendor)
		}
	} else {
		return nil, fmt.Errorf("Device info is nil")
	}

}
func create_amd_gpu_gauge(gpuNum int, gpuInfo *gpu.GraphicsCard) (*prometheus.GaugeVec, error) {
	gauge := get_gpu_gauge_layout(gpuNum, gpuInfo)
	vram, err := get_amd_gpu_vram(gpuInfo)
	if err != nil {
		// return nil, fmt.Errorf("error getting GPU VRAM: %v", err)
		log.Warn("error getting GPU VRAM", zap.Error(err))
	}
	gauge.With(prometheus.Labels{"property": "vram"}).Set(float64(vram))

	return gauge, nil
}
func create_nvidia_gpu_gauge(gpuNum int, gpuInfo *gpu.GraphicsCard) (*prometheus.GaugeVec, error) {
	gauge := get_gpu_gauge_layout(gpuNum, gpuInfo)
	vram, err := get_nvidia_gpu_vram(gpuInfo)
	if err != nil {
		// return nil, fmt.Errorf("error getting GPU VRAM: %v", err)
		log.Warn("error getting GPU VRAM", zap.Error(err))
	}
	gauge.With(prometheus.Labels{"property": "vram"}).Set(float64(vram))

	cudaCores, err := get_nvidia_cuda_cores(gpuInfo)
	if err != nil {
		// return nil, fmt.Errorf("error getting cuda cores count: %v", err)
		log.Warn("error getting cuda cores count", zap.Error(err))
	}
	gauge.With(prometheus.Labels{"property": "cuda_cores"}).Set(float64(cudaCores))

	return gauge, nil
}
func create_intel_gpu_gauge(gpuNum int, gpuInfo *gpu.GraphicsCard) (*prometheus.GaugeVec, error) {
	gauge := get_gpu_gauge_layout(gpuNum, gpuInfo)
	//TODO: implement vram getter for intel gpu, tmp set to 0 to get some value and basic info for gpu
	gauge.With(prometheus.Labels{"property": "vram"}).Set(float64(0))
	return gauge, nil
}

func get_gpu_gauge_layout(gpuNum int, gpuInfo *gpu.GraphicsCard) *prometheus.GaugeVec {
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: fmt.Sprintf("gpu_%v", gpuNum),
		Help: fmt.Sprintf("Device info for %v Model id: \"%v\"", gpuInfo.DeviceInfo.Product.Name, gpuInfo.DeviceInfo.Product.ID),
	}, []string{"property"})
	return gauge
}
