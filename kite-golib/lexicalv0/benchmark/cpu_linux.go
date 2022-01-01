// +build linux

package benchmark

import (
	"os"

	"github.com/shirou/gopsutil/process"
)

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
