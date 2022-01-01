package pythonresource

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testIterations = 2000

// Test_ErrorManager tests that errors returned by the manager does not panic in the caching manager
func Test_ErrorManager(t *testing.T) {
	mgr := NewCachingManager(&errorManager{})
	defer mgr.Close()

	dist := keytypes.Distribution{}
	sym := Symbol{}

	// multiple iterations to force cache usage
	for i := 0; i < 5; i++ {
		time.Sleep(100 * time.Millisecond)

		mgr.Distributions()
		_ = mgr.DistLoaded(dist)

		mgr.TopLevels(dist)
		mgr.NewSymbol(dist, pythonimports.NewPath("dummy"))
		mgr.PathSymbols(kitectx.Background(), pythonimports.NewPath("json", "dumps"))
		mgr.PathSymbol(pythonimports.NewPath("json", "dumps"))
		mgr.ArgSpec(sym)
		mgr.PopularSignatures(sym)
		mgr.CumulativeNumArgsFrequency(sym, 1)
		mgr.KeywordArgFrequency(sym, "o")
		mgr.NumArgsFrequency(sym, 1)
		mgr.Documentation(sym)
		mgr.SymbolCounts(sym)
		mgr.Kwargs(sym)
		mgr.TruthyReturnTypes(sym)
		mgr.TruthyReturnTypes(sym)
		mgr.Kind(sym)
		mgr.Type(sym)
		mgr.Bases(sym)
		mgr.Children(sym)
		mgr.Pkgs()
		mgr.DistsForPkg("dummy-pkg")
		mgr.SigStats(sym)
		mgr.ChildSymbol(sym, "child")
	}
}

// Test_RemoteManager lauches an in-process, remote resourcemanager and
// executes the tests with a rpc client communicating with the server
func Test_RemoteManager(t *testing.T) {
	server, addr, err := StartServer("127.0.0.1:0", false, DefaultOptions)
	require.NoError(t, err)
	defer server.Close()

	client, err := NewRPCClient(addr.String())
	require.NoError(t, err)

	loggingClient := NewLoggingManager(client, false, "rpc_client")
	defer loggingClient.(*loggingManager).printCallStatus("rpc_client")

	cachingClient := NewCachingManager(loggingClient)
	defer cachingClient.Close()

	start := time.Now()
	defer func() {
		fmt.Println("Total test duration:", time.Now().Sub(start).String())
	}()

	for i := 1; i <= testIterations; i++ {
		fmt.Println("Iteration", i)
		client.(*rpcClientManager).testReset()
		testWithManager(t, cachingClient, func() int { return client.(*rpcClientManager).errorCount })
	}
}

// Test_LocalManager executes the tests with the in-process, local resource manager
func Test_LocalManager(t *testing.T) {
	err := datadeps.Enable()
	require.NoError(t, err)

	datadeps.SetLocalOnly()

	mgr, errChan := NewManager(DefaultOptions)
	err = <-errChan
	require.NoError(t, err)

	loggingMgr := NewLoggingManager(mgr, false, "localMgr")
	defer loggingMgr.(*loggingManager).printCallStatus("localMgr")
	defer loggingMgr.Close()

	start := time.Now()
	defer func() {
		fmt.Println("Total test duration:", time.Now().Sub(start).String())
	}()

	for i := 1; i <= testIterations; i++ {
		fmt.Println("Iteration", i)
		testWithManager(t, loggingMgr, func() int { return 0 })
	}
}

func testWithManager(t *testing.T, mgr Manager, errorCount func() int) {
	dists := mgr.Distributions()
	require.Zero(t, errorCount())
	require.NotEmpty(t, dists)

	dist := dists[0]
	_ = mgr.DistLoaded(dist)
	require.Zero(t, errorCount())

	topLevels, err := mgr.TopLevels(dist)
	require.NoError(t, err)
	require.NotEmpty(t, topLevels)
	require.True(t, len(topLevels) >= 10)

	for _, name := range topLevels[0:10] {
		log.Printf("\titerating root symbol %s", name)
		symbol, err := mgr.NewSymbol(dist, pythonimports.NewPath(name))
		require.NoError(t, err)
		require.NotEmpty(t, symbol.Dist().Name, "marshalling has to support Symbol")
	}

	allJSONDumps, err := mgr.PathSymbols(kitectx.Background(), pythonimports.NewPath("json", "dumps"))
	require.NoError(t, err)
	require.EqualValues(t, 0, errorCount())
	require.NotEmpty(t, allJSONDumps)

	jsonDumps, err := mgr.PathSymbol(pythonimports.NewPath("json", "dumps"))
	require.EqualValues(t, 0, errorCount())
	require.NoError(t, err)

	// arg spec
	argSpec := mgr.ArgSpec(jsonDumps)
	require.EqualValues(t, 0, errorCount())
	require.NotNil(t, argSpec)

	// popular signatures
	popularSigs := mgr.PopularSignatures(jsonDumps)
	require.EqualValues(t, 0, errorCount())
	require.NotEmpty(t, popularSigs)

	// cumulative num args frequency
	_, _ = mgr.CumulativeNumArgsFrequency(jsonDumps, 1)
	require.EqualValues(t, 0, errorCount())

	// keyword arg frequency
	_, _ = mgr.KeywordArgFrequency(jsonDumps, "o")
	require.EqualValues(t, 0, errorCount())

	// num arg frequency
	_, _ = mgr.NumArgsFrequency(jsonDumps, 1)
	require.EqualValues(t, 0, errorCount())

	// documentation
	docs := mgr.Documentation(jsonDumps)
	require.EqualValues(t, 0, errorCount())
	assert.NotNil(t, docs)
	assert.NotEmpty(t, docs.Text)

	// symbol counts
	counts := mgr.SymbolCounts(jsonDumps)
	require.EqualValues(t, 0, errorCount())
	assert.NotNil(t, counts)

	// kwards
	kwargs := mgr.Kwargs(jsonDumps)
	require.EqualValues(t, 0, errorCount())
	assert.NotNil(t, kwargs)
	assert.NotEmpty(t, kwargs.Name)
	assert.NotEmpty(t, kwargs.Kwargs)

	// truthy symbols
	truthySymbols := mgr.TruthyReturnTypes(jsonDumps)
	require.EqualValues(t, 0, errorCount())
	require.NotEmpty(t, truthySymbols)

	// return types
	symbolsResp := mgr.TruthyReturnTypes(jsonDumps)
	require.EqualValues(t, 0, errorCount())
	require.NotEmpty(t, symbolsResp)

	// kind
	kind := mgr.Kind(jsonDumps)
	require.EqualValues(t, 0, errorCount())
	require.NotZero(t, kind)

	// type
	typeSymbol, err := mgr.Type(jsonDumps)
	require.NoError(t, err)
	require.EqualValues(t, 0, errorCount())
	require.NotNil(t, typeSymbol)

	// bases
	_ = mgr.Bases(jsonDumps)
	require.EqualValues(t, 0, errorCount())

	// children
	children, err := mgr.Children(jsonDumps)
	require.NoError(t, err)
	require.EqualValues(t, 0, errorCount())
	require.NotEmpty(t, children)

	// canonical symbols
	// this is a very expensive operation, which doesn't seem to be used by kited's features
	/*canonicalSymbols, err := mgr.CanonicalSymbols(jsonDumps.Dist())
	require.NoError(t, err)
	require.EqualValues(t, 0, errorCount())
	require.NotNil(t, canonicalSymbols)*/

	// pkgs
	pkgs := mgr.Pkgs()
	require.EqualValues(t, 0, errorCount())
	require.NotEmpty(t, pkgs)

	// dists
	dists = mgr.DistsForPkg(pkgs[0])
	require.EqualValues(t, 0, errorCount())
	require.NotEmpty(t, dists)
	require.NotEmpty(t, dists[0].Name)

	// SigStats
	sigStats := mgr.SigStats(jsonDumps)
	require.EqualValues(t, 0, errorCount())
	require.NotNil(t, sigStats)

	// child symbol, last due to error status
	childSymbol, err := mgr.ChildSymbol(jsonDumps, "child")
	require.Error(t, err, "child symbol", childSymbol)
	require.EqualValues(t, "attribute child not found", err.Error())
}
