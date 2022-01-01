package livemetrics

import (
	"context"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/performance"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// CPUMetricsFrozen contains all metrics related to CPU usage
type CPUMetricsFrozen struct {
	samples       []float64
	activeSamples []float64
	loadAvg       [][]float64
	fanSpeeds     []float64
	temperatures  []float64
}

type cpuMetrics struct {
	rw            sync.RWMutex
	samples       []float64
	activeSamples []float64
	loadAvg       [][]float64
	fanSpeeds     []float64
	temperatures  []float64
	recorded      bool
	ctxCancel     func()
	lastCaptureTs time.Time
}

func newCPUMetrics() *cpuMetrics {
	ctx, cancel := context.WithCancel(context.Background())
	c := &cpuMetrics{
		ctxCancel: cancel,
	}
	go c.loop(ctx)
	return c
}

func (c *cpuMetrics) close() {
	if c.ctxCancel != nil {
		c.ctxCancel()
		c.ctxCancel = nil
	}
}

func (c *cpuMetrics) zero() {
	c.rw.Lock()
	defer c.rw.Unlock()
	c.zeroLocked()
}

func (c *cpuMetrics) get() *CPUMetricsFrozen {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return c.getLocked(true)
}

func (c *cpuMetrics) dump() *CPUMetricsFrozen {
	c.rw.Lock()
	defer c.rw.Unlock()
	metrics := c.getLocked(false)
	c.zeroLocked()
	return metrics
}

func (c *cpuMetrics) zeroLocked() {
	c.samples = nil
	c.activeSamples = nil
	c.temperatures = nil
	c.fanSpeeds = nil
	c.loadAvg = nil
	c.recorded = false

}

func (c *cpuMetrics) getLocked(copy bool) *CPUMetricsFrozen {

	if copy {
		return &CPUMetricsFrozen{
			samples:       append(make([]float64, 0, len(c.samples)), c.samples...),
			activeSamples: append(make([]float64, 0, len(c.activeSamples)), c.activeSamples...),
			loadAvg:       append(make([][]float64, 0, len(c.loadAvg)), c.loadAvg...),
			fanSpeeds:     append(make([]float64, 0, len(c.fanSpeeds)), c.fanSpeeds...),
			temperatures:  append(make([]float64, 0, len(c.temperatures)), c.temperatures...),
		}
	}
	return &CPUMetricsFrozen{
		samples:       c.samples,
		activeSamples: c.activeSamples,
		loadAvg:       c.loadAvg,
		fanSpeeds:     c.fanSpeeds,
		temperatures:  c.temperatures,
	}
}

// --

var (
	// sample cpu usage every minute
	cpuSampleInterval = time.Minute
)

const (
	maxCPUSamples = 50
)

func (c *cpuMetrics) append(cpu float64, speed float64, temp float64, avg []float64) {
	c.rw.Lock()
	defer c.rw.Unlock()
	s, e := 0, len(c.samples)
	if e > maxCPUSamples {
		// That shouldn't happen but in some case (maybe when there's no connection ar app sleeps) we don't send nor dump
		// the metrics for a long time and samples starts to stack up
		s = e - maxCPUSamples
	}
	c.samples = append(c.samples[s:e], cpu)
	c.fanSpeeds = append(c.fanSpeeds[s:e], speed)
	c.temperatures = append(c.temperatures[s:e], temp)
	if avg != nil {
		// It's safe to take the same range as loadAvg is either always nil or always non nil (OS dependent)
		c.loadAvg = append(c.loadAvg[s:e], avg)
	}
	c.recorded = false
}

func (c *cpuMetrics) recordActive() {
	c.rw.Lock()
	defer c.rw.Unlock()
	if !c.recorded {
		c.activeSamples = append(c.activeSamples, performance.CPUUsage())
		c.recorded = true
	}
}

func (c *cpuMetrics) loop(ctx context.Context) {
	defer func() {
		if ex := recover(); ex != nil {
			rollbar.PanicRecovery(ex)
		}
	}()

	ticker := time.NewTicker(cpuSampleInterval)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			c.append(performance.CPUUsage(), performance.FanSpeed(), performance.CPUTemp(), performance.LoadAvg())

		}
	}
}
