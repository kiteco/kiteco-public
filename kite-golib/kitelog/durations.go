package kitelog

import (
	"bytes"
	"fmt"
	"text/tabwriter"
	"time"
)

type duration struct {
	name     string
	duration time.Duration
}

// Durations tracks durations
type Durations []duration

// Record records a duration
func (t *Durations) Record(name string, d time.Duration) {
	*t = append(*t, duration{name, d})
}

// Flush flushes buffered log lines to the given handler
func (t *Durations) Flush(i Interface) {
	var b bytes.Buffer
	tw := tabwriter.NewWriter(&b, 4, 4, 0, ' ', 0)
	for _, entry := range *t {
		fmt.Fprintf(tw, "   %s\t%s\n", entry.name, entry.duration)
	}
	tw.Flush()

	i.Println(b.String())
}

// WithDurations returns a derived Logger with a new Durations tracker
func (l *Logger) WithDurations() *Logger {
	out := *l
	out.Durations = nil
	return &out
}
