package annotate

import "flag"

var flaskTests bool
var dockerTests bool

func init() {
	flag.BoolVar(&flaskTests, "flask", false, "run tests that use flask")
	flag.BoolVar(&dockerTests, "docker", false, "run tests that use docker")
}
