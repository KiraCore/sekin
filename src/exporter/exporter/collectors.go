package exporter

import (
	"github.com/jaypipes/ghw/pkg/gpu"
	systeminfo "github.com/kiracore/sekin/src/exporter/system_info"
)

func collectTotalCPUCores() error {
	cores := systeminfo.GetTotalCPUCores()
	totalCPUCores.Set(cores)
	return nil
}

func collectTotalRAM() error {
	ram, err := systeminfo.GetTotalRAM()
	if err != nil {
		return err
	}
	totalRAM.Set(ram)
	return nil
}

func collectTotalDiskSpace() error {
	space, err := systeminfo.GetTotalDiskSpace()
	if err != nil {
		return err
	}
	totalDiskSpace.Set(space)
	return nil
}

func collectTotalBandwidth() error {
	downloadSpeed, uploadSpeed, err := systeminfo.GetTotalBandwidth()
	if err != nil {
		return err
	}
	downloadBandwidth.Set(downloadSpeed)
	uploadBandwidth.Set(uploadSpeed)
	return nil
}

func collectTotalCPUGHz() error {
	totalGhz, err := systeminfo.GetTotalCPUGHz()
	if err != nil {
		return err
	}
	totalCPUGHz.Set(totalGhz)
	return nil
}

func get_amd_gpu_vram(gpuInfo *gpu.GraphicsCard) (float64, error) {
	return systeminfo.GetAmdGpuVram(gpuInfo.Address)
}
