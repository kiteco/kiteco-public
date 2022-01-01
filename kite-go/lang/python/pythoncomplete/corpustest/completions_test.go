// +build !slow

package test

import (
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-golib/complete/corpustests"
)

func TestFastCorpusTests(t *testing.T) {
	runFromCorpus(t, 30*time.Second, corpustests.OkState)
}

func TestRequestPartialGet(t *testing.T) {
	runOneTestFromCorpus(t, 30*time.Minute, "ggnn_subtoken.py", "test_ggnn_partial_from_attr", corpustests.OkState)
}

/*
Example of usage to trigger one specific test for testing and debugging
func TestRequestPartialGet(t *testing.T) {
	runOneTestFromCorpus(t, 30*time.Minute, "ggnn_subtoken.py", "test_partial_get", corpustests.OkState)
}*/
