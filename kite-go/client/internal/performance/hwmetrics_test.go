package performance

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/shirou/gopsutil/host"

	"github.com/stretchr/testify/assert"
)

func Test_testCPUTemp(t *testing.T) {
	temps, _ := host.SensorsTemperatures()
	// We don't check for errors as Windows can returns Not Supported error on travis
	fmt.Println("CPU Temp: ", temps)
	for _, temp := range temps {
		// skip weird temps for _min and _max values
		if !strings.HasSuffix(temp.SensorKey, "_min") && !strings.HasSuffix(temp.SensorKey, "_max") {
			assert.True(t, temp.Temperature >= 0, "The temperatures should be >= 0 (%s temp is %v)", temp.SensorKey, temp.Temperature)
		}
	}
}

func Test_fanSpeedInternal(t *testing.T) {
	speeds, err := fanSpeedsImpl()
	fmt.Println("Fan Speeds: ", speeds)
	assert.NoError(t, err)
	for _, s := range speeds {
		assert.True(t, s.Speed >= 0, "The speeds should be >= 0 (%s speed is %v)", s.SensorKey, s.Speed)
	}
}

func Test_loadAvg(t *testing.T) {
	loads := LoadAvg()
	time.Sleep(10 * time.Second)
	fmt.Println("LoadAvg (1m, 5m, 15m):", loads)
	// Windows doesn't return any load avg, the loop will just be skipped in this case
	for _, l := range loads {
		assert.True(t, l >= 0.0, "All loads should be >= 0")
	}
}
