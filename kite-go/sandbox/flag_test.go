package sandbox

import "flag"

var (
	dockerTests bool
)

func init() {
	flag.BoolVar(&dockerTests, "docker", false, "run tests that use docker")
}
