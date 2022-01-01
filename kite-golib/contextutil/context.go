package contextutil

import (
	"context"
	"io"
	"log"
	"time"
)

type key int

const (
	loggerKey    key = 0
	durationsKey key = 1
)

// NewContext creates a new context that contains the input logger.
func NewContext(logger *log.Logger) context.Context {
	return WithLogger(context.Background(), logger)
}

// WithLogger creates a new context that contains the input logger.
func WithLogger(ctx context.Context, logger *log.Logger) context.Context {
	ctx = context.WithValue(ctx, loggerKey, logger)
	ctx = context.WithValue(ctx, durationsKey, &durations{})
	return ctx
}

// LoggerFromContext returns the logger from the context.
func LoggerFromContext(ctx context.Context) *log.Logger {
	logger, _ := ctx.Value(loggerKey).(*log.Logger)
	return logger
}

// --

// RecordDuration records a (name, duration) entry
func RecordDuration(ctx context.Context, name string, d time.Duration) {
	if durations, ok := ctx.Value(durationsKey).(*durations); ok {
		durations.add(name, d)
	}
}

// ResetDurations resets durations
func ResetDurations(ctx context.Context) {
	if durations, ok := ctx.Value(durationsKey).(*durations); ok {
		durations.reset()
	}
}

// FprintDurations prints out recorded durations to the provided io.Writer.
func FprintDurations(ctx context.Context, w io.Writer) {
	if durations, ok := ctx.Value(durationsKey).(*durations); ok {
		durations.fprint(w)
	}
}
