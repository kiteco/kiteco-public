package event

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("events")

	flushDuration = section.SampleDuration("Time per block flush")
	totalDuration = section.SampleDuration("Total event journal processsing")

	eventsPerBlock  = section.SampleInt64("Events per block")
	retriesPerFlush = section.SampleInt64("Flush retries per block")

	bytesPerBlock = section.SampleByte("Bytes per block (compressed)")
)

func init() {
	// These events happen infrequently, so its OK to sample them at 100%
	bytesPerBlock.SetSampleRate(1.0)
	retriesPerFlush.SetSampleRate(1.0)
	eventsPerBlock.SetSampleRate(1.0)
	flushDuration.SetSampleRate(1.0)
}
