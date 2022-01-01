package main

import "fmt"

type logLevel int

const (
	logLevelNone logLevel = iota
	logLevelVerbose
	logLevelWarn
	logLevelSevere
)

func logf(l logLevel, format string, args ...interface{}) {
	if logger != nil && l >= level {
		switch level {
		case logLevelSevere:
			format = "SEVERE: " + format
		case logLevelWarn:
			format = "WARN: " + format
		}
		fmt.Fprintf(logger, format, args...)
	}
}
