package awsutil

import "flag"

// Some tests in this package rely on having a network connection and credentials for
// AWS so should not be part of the CI process. But if you modify the cache logic then
// you should run "go test -aws" to run these tests.

var awsTests bool

func init() {
	flag.BoolVar(&awsTests, "aws", false, "run tests that rely on AWS connectivity and credentials")
}
