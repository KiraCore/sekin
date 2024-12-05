package types

const (
	DEVICES_BASE_PATH  string = "/sys/bus/pci/devices"
	AMO_VRAM_FILE_NAME string = "mem_info_vram_total"
)

// Use cautiously, when comparing strings some vendors or utils type in different casing
const (
	VENDOR_AMD_GPU_ID    = "1002"
	VENDOR_NVIDIA_GPU_ID = "10DE"
	VENDOR_INTEL_GPU_ID  = "8086"
)
