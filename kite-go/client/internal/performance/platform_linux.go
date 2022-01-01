// +build linux

package performance

import (
	"fmt"
	"os"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/process"
)

// memoryUsage returns the current resident memory size in bytes
func memoryUsage() int64 {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return -1.0
	}

	memInfo, err := p.MemoryInfo()
	if err != nil || memInfo == nil {
		return -1.0
	}
	return int64(memInfo.RSS)
}

// osVersion returns the OS version as a string
func osVersion() string {
	_, family, version, err := host.PlatformInformation()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s", family, version)
}

// cpuUsage gets the current CPU usage
func cpuUsage() float64 {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return -1.0
	}

	cpu, err := p.CPUPercent()
	if err != nil {
		return -1.0
	}
	return cpu
}
