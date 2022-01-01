package words

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireAggregator(t *testing.T, tmpdir string, max int) *Aggregator {
	agg, err := NewAggregator(tmpdir)
	require.NoError(t, err)
	agg.maxWordCount = max
	return agg
}

func Test_Aggregator(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "wordcount-aggregator")
	require.NoError(t, err)

	maxCount := 10
	agg := requireAggregator(t, tmpDir, maxCount)

	wordCount := 5
	numWords := 100

	for i := 0; i < numWords; i++ {
		counts := make(Counts)
		counts.Hit(fmt.Sprintf("word%d", i), ".py", wordCount)
		agg.Add(counts)
	}

	err = agg.Flush()
	require.NoError(t, err)

	merged, err := agg.Merge(0)
	require.NoError(t, err)

	require.Equal(t, numWords, len(merged))
	for _, ext := range merged {
		require.Equal(t, wordCount, ext.Sum())
	}
}
