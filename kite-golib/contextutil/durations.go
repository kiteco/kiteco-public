package contextutil

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"
)

type entry struct {
	name     string
	duration time.Duration
}

type durations struct {
	entries []entry
}

func (t *durations) reset() {
	t.entries = nil
}

func (t *durations) add(name string, d time.Duration) {
	t.entries = append(t.entries, entry{name, d})
}

func (t *durations) fprint(w io.Writer) {
	tw := tabwriter.NewWriter(w, 4, 4, 0, ' ', 0)
	for _, entry := range t.entries {
		fmt.Fprintf(tw, "   %s\t%s\n", entry.name, entry.duration)
	}
	tw.Flush()
}
