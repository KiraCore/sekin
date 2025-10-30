package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

var (
	lastCPUSample   time.Time
	lastCPUUsage    float64
	lastMemoryUsage float64
	cpuUsageMutex   sync.Mutex
)

func GetCPUUsage() float64 {
	cpuUsageMutex.Lock()
	defer cpuUsageMutex.Unlock()

	if time.Since(lastCPUSample) < time.Second {
		return lastCPUUsage
	}

	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return lastCPUUsage
	}

	if len(percent) > 0 {
		lastCPUUsage = percent[0]
		lastCPUSample = time.Now()
	}

	return lastCPUUsage
}

func GetMemoryUsage() float64 {
	v, err := mem.VirtualMemory()
	if err != nil {
		return lastMemoryUsage
	}

	lastMemoryUsage = v.UsedPercent

	return lastMemoryUsage
}

func GenerateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		now := time.Now().UnixNano()
		return fmt.Sprintf("fallback-%x", now)
	}
	return hex.EncodeToString(b)
}

func MapFloatMinValue(m map[string]float64) (string, float64, error) {
	if len(m) == 0 {
		return "", 0, fmt.Errorf("cannot find minimum in empty map")
	}

	var minKey string
	var minValue float64
	firstItem := true

	for key, value := range m {
		if firstItem {
			minKey = key
			minValue = value
			firstItem = false
			continue
		}

		if value < minValue {
			minKey = key
			minValue = value
		}
	}

	return minKey, minValue, nil
}
