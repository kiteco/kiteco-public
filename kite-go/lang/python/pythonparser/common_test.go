package pythonparser

import "flag"

var opts Options

func init() {
	flag.BoolVar(&opts.Trace, "traceparse", false, "turn on parser tracing")
}
