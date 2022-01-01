package sandbox

import "time"

// Limits represents limits on the duration and output of a subprocess.
type Limits struct {
	Timeout  time.Duration `yaml:"timeout"`
	MaxLines int           `yaml:"max_output_lines"`
	MaxBytes int           `yaml:"max_output_bytes"`
}

// DefaultLimits defines a standard set of time and output limits for python code examples
var DefaultLimits = &Limits{
	Timeout:  10 * time.Second,
	MaxLines: 100,
	MaxBytes: 100000,
}

// makeTimeoutChannel constructs a channel that posts a message after the timeout, or
// a channel that never posts a message if there is no timeout.
func (l *Limits) makeTimeoutChannel() <-chan time.Time {
	if l.Timeout == 0 {
		return make(<-chan time.Time)
	}
	return time.After(l.Timeout)
}

// Exceeded tests whether the limits are exceeded by the specified number of lines and bytes
func (l *Limits) exceeded(lines, bytes int) bool {
	return (l.MaxLines > 0 && lines > l.MaxLines) || (l.MaxBytes > 0 && bytes > l.MaxBytes)
}
