// +build linux

package performance

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/load"
)

func loadAvgImpl() []float64 {
	result := make([]float64, 3)
	avg, err := load.Avg()
	if err != nil {
		return nil
	}
	result[0] = avg.Load1
	result[1] = avg.Load5
	result[2] = avg.Load15
	return result
}

// Code inspired from SensorsTemperaturesWithContext in host_linux.go from gopsutil
func fanSpeedsImpl() ([]FanSpeedStat, error) {
	var fanSpeeds []FanSpeedStat
	files, err := filepath.Glob("/sys/class/hwmon/hwmon*/fan*_*")
	if err != nil {
		return fanSpeeds, err
	}
	if len(files) == 0 {
		// CentOS has an intermediate /device directory:
		files, err = filepath.Glob("/sys/class/hwmon/hwmon*/device/fan*_*")
		if err != nil {
			return fanSpeeds, err
		}
	}

	for _, file := range files {
		filename := strings.Split(filepath.Base(file), "_")
		if filename[1] == "label" {
			// Do not try to read the temperature of the label file
			continue
		}

		// Get the label of the temperature you are reading
		var label string
		c, _ := ioutil.ReadFile(filepath.Join(filepath.Dir(file), filename[0]+"_label"))
		if c != nil {
			//format the label from "Core 0" to "core0_"
			label = fmt.Sprintf("%s_", strings.Join(strings.Split(strings.TrimSpace(strings.ToLower(string(c))), " "), ""))
		}

		// Get the name of the tempearture you are reading
		name, err := ioutil.ReadFile(filepath.Join(filepath.Dir(file), "name"))
		if err != nil {
			return fanSpeeds, err
		}

		// Get the temperature reading
		current, err := ioutil.ReadFile(file)
		if err != nil {
			return fanSpeeds, err
		}
		spd, err := strconv.ParseFloat(strings.TrimSpace(string(current)), 64)
		if err != nil {
			continue
		}

		spdName := strings.TrimSpace(strings.ToLower(string(strings.Join(filename[1:], ""))))
		fanSpeeds = append(fanSpeeds, FanSpeedStat{
			SensorKey: fmt.Sprintf("%s_%s%s", strings.TrimSpace(string(name)), label, spdName),
			Speed:     spd,
		})
	}
	return fanSpeeds, nil
}
