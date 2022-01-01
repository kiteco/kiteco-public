//+build windows

package performance

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-golib/rollbar"

	"github.com/lxn/win"
)

const (
	loadAvgSampleTime = 5 * time.Second
	counterPath       = "\\System\\Processor Queue Length"
)

var (
	handle        win.PDH_HQUERY
	counterHandle win.PDH_HCOUNTER
	mutex         sync.Mutex

	load1, load5, load15 float64

	expFactor1  = 1.0 / math.Exp(float64(loadAvgSampleTime)/float64(1*time.Minute))
	expFactor5  = 1.0 / math.Exp(float64(loadAvgSampleTime)/float64(5*time.Minute))
	expFactor15 = 1.0 / math.Exp(float64(loadAvgSampleTime)/float64(15*time.Minute))
)

func init() {
	ret := win.PdhOpenQuery(0, 0, &handle)
	fmt.Printf("Open Query return code is %x\n", ret)
	ret = win.PdhAddEnglishCounter(handle, counterPath, 0, &counterHandle)
	fmt.Printf("Add Counter return code is %x\n", ret)
	closer := make(chan struct{})
	go loopLoadAvg(closer)

}

func loopLoadAvg(closer chan struct{}) {
	defer func() {
		if ex := recover(); ex != nil {
			rollbar.PanicRecovery(ex)
		}
	}()

	ticker := time.NewTicker(loadAvgSampleTime)
	for {
		select {
		case <-closer:
			ticker.Stop()
			return
		case <-ticker.C:
			func() {
				instantLoad := getProcessorQueueLengthCounter()
				mutex.Lock()
				load1 = load1*expFactor1 + instantLoad*(1-expFactor1)
				load5 = load5*expFactor5 + instantLoad*(1-expFactor5)
				load15 = load15*expFactor15 + instantLoad*(1-expFactor15)
				defer mutex.Unlock()
			}()
		}
	}
}

func getProcessorQueueLengthCounter() float64 {
	var derp win.PDH_FMT_COUNTERVALUE_DOUBLE
	win.PdhCollectQueryData(handle)
	win.PdhGetFormattedCounterValueDouble(counterHandle, nil, &derp)
	return derp.DoubleValue
}

func loadAvgImpl() []float64 {
	mutex.Lock()
	defer mutex.Unlock()
	return append([]float64{}, load1, load5, load15)
}
