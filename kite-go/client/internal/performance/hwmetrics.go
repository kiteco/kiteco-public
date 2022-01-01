package performance

import (
	"strings"

	"github.com/shirou/gopsutil/host"
)

// FanSpeed returns the average fanSpeed in RPM (or -1 if no fan telemetry is available)
func FanSpeed() float64 {
	speeds, err := fanSpeedsImpl()
	if err != nil {
		return -1.0
	}
	var result float64
	var count int32
	if len(speeds) == 1 {
		return speeds[0].Speed
	}
	for _, r := range speeds {
		count++
		result += r.Speed
	}

	if count == 0 {
		return -1.0
	}
	return result / float64(count)
}

// FanSpeedStat contains information returned by OS for fan speeds
type FanSpeedStat struct {
	SensorKey string
	Speed     float64 // Speed is in RPM
}

// CPUTemp returns the current temperature of the CPU
// Temperature is only available with Administrator privileges on Windows
func CPUTemp() float64 {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		return -1.0
	}
	var result float64
	var count int32
	if len(temps) == 1 {
		return temps[0].Temperature
	}

	for _, r := range temps {
		if strings.Contains(r.SensorKey, "input") ||
			strings.Contains(r.SensorKey, "TC0P") ||
			strings.Contains(r.SensorKey, "ThermalZone") {
			count++
			result += r.Temperature
		}
	}
	if count == 0 {
		return -1.0
	}
	return result / float64(count)
}

// LoadAvg return the average cpu load over the last 1, 5 and 15 minutes
// Warning, for windows this value is emulated by kited, so kited needs to be running since 15 min
// for the 15 min load average to be valid.
func LoadAvg() []float64 {
	return loadAvgImpl()
}
