// +build slow

package performancetest

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ProviderThresholds struct {
	FailureRatio float32
	Thresholds   map[string]time.Duration
}

var (
	thresholds = ProviderThresholds{
		FailureRatio: 2,
		Thresholds: map[string]time.Duration{
			"CallModel":            376 * time.Millisecond,
			"CallPatterns":         162 * time.Microsecond,
			"GGNNModelAccumulator": 590 * time.Millisecond,
		},
	}
)

var providerTimeout = 10 * time.Minute

func Test_AllFiles(t *testing.T) {
	err := datadeps.Enable()
	require.NoError(t, err)

	pythonproviders.SetUseGGNNCompletions(true)

	allFiles, err := filepath.Glob("./tests/*.py")
	require.NoError(t, err)

	mgr, errC := pythonresource.NewManager(pythonresource.DefaultLocalOptions)
	require.NoError(t, <-errC)
	aggregatedStats := make(map[string]time.Duration)

	for _, file := range allFiles {
		absFile, err := filepath.Abs(file)
		require.NoError(t, err)

		filename := filepath.Base(file)
		t.Run(filename, func(t *testing.T) {
			stats, err := TestProviders(mgr, absFile)
			require.NoError(t, err)

			// no provider must take longer than providerTimeout
			for _, stat := range stats {
				aggregatedStats[stat.Name] += stat.TotalDuration()
				assert.True(t,
					stat.TotalDuration() <= providerTimeout,
					"provider must not take longer than %s to compute completions, time taken: %s",
					providerTimeout.String(), stat.TotalDuration().String())
			}

			// fixme test total of all providers?
		})
	}
	t.Run("Provider Performance check", func(t *testing.T) {

		testProviderPerformances(aggregatedStats, t)
	})
}

func testProviderPerformances(stats map[string]time.Duration, t *testing.T) {
	fmt.Println(stats)
	var errors []string

	for name, threshold := range thresholds.Thresholds {
		duration, ok := stats[name]
		if !ok {
			assert.FailNowf(t, "", "Error, the performance test threshold file contains an entry for a provider that hasn't been executed : %s", name)
		}
		if duration > time.Duration(float32(threshold)*thresholds.FailureRatio) {
			errors = append(errors, fmt.Sprintf("- The provider %s went too slow (%v and the threshold for it is %v)", name, duration, threshold))
			thresholds.Thresholds[name] = duration
		}
		if threshold > time.Duration(float32(duration)*thresholds.FailureRatio) {
			errors = append(errors, fmt.Sprintf("- The provider %s went too fast (%v and the threshold for it is %v)", name, duration, threshold))
			thresholds.Thresholds[name] = duration
		}
	}
	if len(errors) > 0 {
		assert.FailNowf(t, "One (or more) provider failed performance test", "%s\n\nPlease fix them or update the content of the thresholds variable (in kite-go/lang/python/pythoncomplete/performancetest/performance_test.go)", strings.Join(errors, "\n"))
	}
}
