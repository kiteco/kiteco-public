package localcodetests

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/stretchr/testify/require"
)

func getPandasManager(t *testing.T) pythonresource.Manager {
	require.NoError(t, datadeps.UseAssetFileMap())

	// load resource manager with only the following distributions
	opts := pythonresource.DefaultLocalOptions
	opts.CacheSize = 0
	opts.Dists = []keytypes.Distribution{
		keytypes.PandasDistribution,
	}
	rm, errc := pythonresource.NewManager(opts)
	require.NoError(t, <-errc)

	return rm

}

// TestPandasResolution tests that values returned when accessing a DataFrame are correct
func TestPandasResolution(t *testing.T) {
	manager := getPandasManager(t)
	classDef := `
import pandas

df = pandas.DataFrame()
df["test"]= 5
blap = df["test"]
bloup = df[["test", "nlip"]]
blop = df[df["age"] > 18]
	`

	assertResolveOpts(t, opts{
		src:     classDef,
		srcpath: "/code/classDef.py",
		localfiles: map[string]string{
			"/code/classDef.py": classDef,
		},
		expected: map[string]string{
			"blap":  "externalinstance:pandas.core.series.Series",
			"bloup": "DataFrame {<nil>: <nil>} []", // Default dataframe as columns have changed
			"blop":  "DataFrame {str: externalinstance:pandas.core.series.Series} [\"test\"]"},
		manager: manager,
	})
}
