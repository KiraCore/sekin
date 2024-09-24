package prometheusexporter

import (
	systeminfo "github.com/kiracore/sekin/src/shidai/internal/utils/system_info"
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

func collectTotalGPUCUDACores() error {
	cudaCores, err := systeminfo.GetTotalGPUCUDACores()
	if err != nil {
		return err
	}
	totalGPUCUDACores.Set(cudaCores)
	return nil
}

func collectTotalVRAM() error {
	vram, err := systeminfo.GetTotalVRAM()
	if err != nil {
		return err
	}
	totalVRAM.Set(vram)
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
