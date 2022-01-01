package performancetest

import (
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/driver"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// TestEnv combines Global and Inputs
type TestEnv struct {
	Global pythonproviders.Global
	Inputs pythonproviders.Inputs
}

func requireSelectedBuffer(src string) (data.SelectedBuffer, error) {
	var sb data.SelectedBuffer
	switch parts := strings.Split(src, "$"); len(parts) {
	case 1:
		sb = data.NewBuffer(src).Select(data.Cursor(len(src)))
	case 2:
		sb = data.NewBuffer(strings.Join(parts, "")).Select(data.Cursor(len(parts[0])))
	default:
		return sb, errors.Errorf("bad test case source, expect 1 or 2 parts, got %d for: %s", len(parts), src)
	}
	return sb, nil
}

// SetupTestEnv returns a new test environment for a given file path
func SetupTestEnv(mgr pythonresource.Manager, dataFilePath string) (TestEnv, error) {
	if _, err := os.Stat(dataFilePath); err != nil {
		return TestEnv{}, err
	}

	contentBytes, err := ioutil.ReadFile(dataFilePath)
	if err != nil {
		return TestEnv{}, err
	}

	models, err := pythonmodels.New(pythonmodels.DefaultOptions)
	if err != nil {
		return TestEnv{}, err
	}

	global := pythonproviders.Global{
		ResourceManager: mgr,
		FilePath:        dataFilePath,
		Models:          models,
		Product:         licensing.Pro,
	}

	sb, err := requireSelectedBuffer(string(contentBytes))
	if err != nil {
		return TestEnv{}, err
	}

	inputs, err := pythonproviders.NewInputs(kitectx.Background(), global, sb, false, false)
	return TestEnv{
		Global: global,
		Inputs: inputs,
	}, err
}

// TestProviders runs all completion providers on a given file and returns statistics
func TestProviders(mgr pythonresource.Manager, testFilePath string) ([]*ProviderStats, error) {
	env, err := SetupTestEnv(mgr, testFilePath)
	if err != nil {
		return nil, err
	}

	var statList []*ProviderStats
	for _, p := range driver.TestProviders() {
		// warm up before recording the stats
		for i := 0; i < 2; i++ {
			profileProvider(env, p, testFilePath)
		}
		statList = append(statList, profileProvider(env, p, testFilePath))
	}

	// sort fast to slow
	sort.Slice(statList, func(i, j int) bool {
		return statList[i].TotalDuration() < statList[j].TotalDuration()
	})
	return statList, nil
}

// ProfileProvider
func profileProvider(env TestEnv, provider pythonproviders.Provider, testFilePath string) *ProviderStats {
	stats := ProviderStats{
		Name:   reflect.TypeOf(provider).Name(),
		Source: testFilePath,
	}

	stats.Start()
	_ = provider.Provide(kitectx.Background(), env.Global, env.Inputs, func(ctx kitectx.Context, sb data.SelectedBuffer, mc pythonproviders.MetaCompletion) {
		stats.Add(mc.Snippet.Text)
	})
	stats.Stop()
	return &stats
}
