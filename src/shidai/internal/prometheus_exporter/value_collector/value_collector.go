package valuecollector

import "github.com/shirou/gopsutil/cpu"

func Get_TotalGhz() (float64, error) {
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
