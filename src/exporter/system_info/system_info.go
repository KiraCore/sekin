package systeminfo

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/gpu"
	"github.com/kiracore/sekin/src/exporter/logger"
	"github.com/kiracore/sekin/src/exporter/types"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/showwin/speedtest-go/speedtest"
	"go.uber.org/zap"
)

var log = logger.GetLogger()

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
//
// returns bytes per second
//
//	return download, upload, error
func GetTotalBandwidth() (float64, float64, error) {
	log.Debug("Testing internet bandwidth.")
	var speedtestClient = speedtest.New()
	serverList, err := speedtestClient.FetchServers()
	if err != nil {
		log.Debug("error when fetching servers", zap.Error(err))
		return 0, 0, err
	}
	targets, err := serverList.FindServer([]int{})
	if err != nil {
		log.Debug("error when finding servers", zap.Error(err))
		return 0, 0, err
	}

	var uploadSpeed, downloadSpeed float64
	for _, s := range targets {
		// Please make sure your host can access this test server,
		// otherwise you will get an error.
		// It is recommended to replace a server at this time
		err = s.DownloadTest()
		if err != nil {
			log.Debug("error when testing download speed", zap.Error(err))
			return 0, 0, err
		}
		err = s.UploadTest()
		if err != nil {
			log.Debug("error when testing upload speed", zap.Error(err))
			return 0, 0, err
		}
		log.Debug("speed test", zap.Any("server", s), zap.Float64("DownloadTest", float64(s.DLSpeed)), zap.Float64("UploadTest", float64(s.ULSpeed)))
		uploadSpeed = float64(s.ULSpeed)
		downloadSpeed = float64(s.DLSpeed)
		// Note: The unit of s.DLSpeed, s.ULSpeed is bytes per second, this is a float64.
		s.Context.Reset() // reset counter
	}
	return downloadSpeed, uploadSpeed, nil
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

func GetAmdGpuVram(gpuAddress string) (float64, error) {
	gpuPath := filepath.Join(types.DEVICES_BASE_PATH, gpuAddress)
	vramPath := filepath.Join(gpuPath, types.AMO_VRAM_FILE_NAME)
	vramContent, err := os.ReadFile(vramPath)
	if err != nil {
		return 0, err
	}
	vram, err := strconv.ParseFloat(strings.Replace(string(vramContent), "\n", "", -1), 64)
	if err != nil {
		return 0, err
	}
	return vram, nil
}

// Collects all available gpus info on the system
func CollectGpusInfo() ([]*gpu.GraphicsCard, error) {
	gpu, err := ghw.GPU()
	if err != nil {
		return nil, err
	}
	return gpu.GraphicsCards, nil
}

func GetNvidiaGpuVram(gpuAddress string) (vram float64, err error) {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return 0, fmt.Errorf("unable to initialize NVML: %w", ret)
	}
	defer func() {
		ret := nvml.Shutdown()
		if ret != nvml.SUCCESS {
			err = fmt.Errorf("unable to shutdown NVML: %w", ret)
		}
	}()
	device, ret := nvml.DeviceGetHandleByPciBusId(gpuAddress)
	if ret != nvml.SUCCESS {
		return 0, ret
	}
	memInfo, ret := device.GetMemoryInfo()
	if ret != nvml.SUCCESS {
		return 0, ret
	}
	return float64(memInfo.Total), nil
}

func GetNvidiaCudaCores(gpuAddress string) (cudaCores float64, err error) {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return 0, fmt.Errorf("unable to initialize NVML: %w", ret)
	}
	defer func() {
		ret := nvml.Shutdown()
		if ret != nvml.SUCCESS {
			err = fmt.Errorf("unable to shutdown NVML: %w", ret)
		}
	}()
	device, ret := nvml.DeviceGetHandleByPciBusId(gpuAddress)
	if ret != nvml.SUCCESS {
		return 0, ret
	}
	cuda, ret := device.GetNumGpuCores()
	if ret != nvml.SUCCESS {
		return 0, ret
	}
	return float64(cuda), nil
}
