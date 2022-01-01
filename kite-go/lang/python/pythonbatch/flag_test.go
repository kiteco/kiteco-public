package pythonbatch

import "flag"

var (
	docProcessTests bool
)

func init() {
	flag.BoolVar(&docProcessTests, "docs", false, "run tests for DocProcess")
}
