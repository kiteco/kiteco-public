package cpuinfo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/klauspost/cpuid"
)

var vendorMapping = map[cpuid.Vendor]string{
	cpuid.AMD:       "AMD",
	cpuid.Bhyve:     "Bhyve",
	cpuid.Hygon:     "Hygon",
	cpuid.Intel:     "Intel",
	cpuid.KVM:       "KVM",
	cpuid.MSVM:      "MSVM",
	cpuid.NSC:       "NSC",
	cpuid.Transmeta: "Transmeta",
	cpuid.VIA:       "VIA",
	cpuid.VMware:    "VMware",
	cpuid.XenHVM:    "XenHVM",
}

// CPUInfo defines properties of the current CPU and is support JSON marshalling and unmarshalling
type CPUInfo struct {
	VendorID      string   `json:"vendor"`
	BrandName     string   `json:"brand"`
	Family        int      `json:"family"`
	Model         int      `json:"model"`
	PhysicalCores int      `json:"cores"`
	LogicalCores  int      `json:"threads"`
	Features      []string `json:"flags"`
}

// String implements fmt.Stringer, it returns most of the cpu properties in a string
func (c CPUInfo) String() string {
	return fmt.Sprintf("vendor %s, brand %s, family %d, model %d, %d cores, %d threads, flags: %s", c.VendorID, c.BrandName, c.Family, c.Model, c.PhysicalCores, c.LogicalCores, strings.Join(c.Features, ","))
}

// Get returns information about the machine's cpu
func Get() CPUInfo {
	cpu := cpuid.CPU
	sortedFlags := cpu.Features.Strings()
	sort.Strings(sortedFlags)

	return CPUInfo{
		BrandName:     cpu.BrandName,
		VendorID:      vendorMapping[cpu.VendorID],
		Family:        cpu.Family,
		Model:         cpu.Model,
		Features:      sortedFlags,
		PhysicalCores: cpu.PhysicalCores,
		LogicalCores:  cpu.LogicalCores,
	}
}
