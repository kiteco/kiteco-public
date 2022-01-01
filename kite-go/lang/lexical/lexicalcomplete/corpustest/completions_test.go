// +build !slow

package corpustest

import (
	"testing"
	"time"
)

func TestFastCorpusTests(t *testing.T) {
	runFromCorpus(t, 30*time.Second, "ok")
}
