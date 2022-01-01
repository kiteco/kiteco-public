package performance

// MemoryUsage returns the amount of memory that the menubar is currently using
func MemoryUsage() int64 {
	return memoryUsage()
}

// OsVersion returns the OS version as a string
func OsVersion() string {
	return osVersion()
}

// CPUUsage gets the current CPU usage
func CPUUsage() float64 {
	return cpuUsage()
}
