package pythonstatic

import "flag"

// Long running tests

var skeletonTests bool

func init() {
	flag.BoolVar(&skeletonTests, "skeletons", false, "run long running skeleton tests")
}
