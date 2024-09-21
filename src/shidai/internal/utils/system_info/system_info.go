package systeminfo

import (
	"runtime"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

// GetTotalCPUGHz returns the total CPU GHz available (sum of max frequencies of all cores).
func GetTotalCPUGHz() (float64, error) {
	info, err := cpu.Info()
	if err != nil {
		return 0, err
	}

	totalGHz := 0.0
	for _, cpuInfo := range info {
		totalGHz += float64(cpuInfo.Mhz) / 1000.0 // Convert MHz to GHz
	}

	return totalGHz, nil
}

func GetTotalCPUCores() float64 {
	return float64(runtime.NumCPU())
}

// GetTotalRAM returns the total amount of RAM available in bytes.
func GetTotalRAM() (float64, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return float64(v.Total), nil
}

// GetTotalDiskSpace returns the total disk space available in bytes.
func GetTotalDiskSpace() (float64, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return 0, err
	}
	var total uint64 = 0
	for _, p := range partitions {
		usage, _ := disk.Usage(p.Mountpoint)
		total += usage.Total
	}
	return float64(total), nil
}

// GetTotalBandwidth returns the total available bandwidth in bits per second.
func GetTotalBandwidth() (float64, error) {
	// TODO: need more advanced logic for this, need reconsider if we need this
	return 0, nil
}

// GetTotalGPUCUDACores returns the total number of GPU CUDA cores available.
func GetTotalGPUCUDACores() (float64, error) {
	// TODO: need to check each case for each gpu manufactures
	return 0, nil
}

// GetTotalVRAM returns the total VRAM available in bytes.
func GetTotalVRAM() (float64, error) {
	// TODO: need to check each case for each gpu manufactures
	return 0, nil
}
