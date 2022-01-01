// +build slow

package test

import (
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-golib/complete/corpustests"
)

func TestSlowCorpusTests(t *testing.T) {
	runFromCorpus(t, 30*time.Minute, corpustests.SlowState)
}
