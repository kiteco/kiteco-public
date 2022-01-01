//go:generate go-bindata -pkg pythonexpr shards.json

package pythonexpr

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"sync/atomic"
	"time"

	"github.com/kiteco/kiteco/kite-go/kitestatus"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

const (
	checkModelsDuration = time.Minute
	modelUsedTimeout    = 5 * time.Minute
)

var (
	// exprModelShards is a slice of shards, one per cluster of packages
	exprModelShards = mustReadShards()

	// kite_status metric to count how many shards are loaded
	loadedExprShards = kitestatus.GetCounter("expr_shards_loaded")
)

// ExprModelShards returns a slice of shards, one per cluster of packages
func ExprModelShards() []Shard {
	shards := make([]Shard, len(exprModelShards))
	copy(shards, exprModelShards)
	return shards
}

// ShardsFromFile reads shards from the provided path
func ShardsFromFile(path string) ([]Shard, error) {
	r, err := fileutil.NewReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var shards []Shard
	err = json.NewDecoder(r).Decode(&shards)
	if err != nil {
		return nil, err
	}

	return shards, nil
}

// ShardsFromModelPath constructs a slice of shards, used for some backwards compatability
func ShardsFromModelPath(path string) []Shard {
	return []Shard{{ModelPath: path}}
}

func mustReadShards() []Shard {
	var shards []Shard
	buf, err := Asset("shards.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(buf, &shards)
	if err != nil {
		panic(err)
	}
	return shards
}

// selectShard will take a slice of imports and return the best matching shard
func selectShard(shards []Shard, importMap map[string]bool) int32 {
	type shardAndCount struct {
		shardIdx int32
		count    int
	}

	// Count how many imports are in each shard
	var counts []shardAndCount
	for idx, shard := range shards {
		var count int
		for _, pkg := range shard.Packages {
			if _, ok := importMap[pkg]; ok {
				count++
			}
		}
		counts = append(counts, shardAndCount{
			shardIdx: int32(idx),
			count:    count,
		})
	}

	// Sort to select the one with the most imports. Note, we use SliceStable
	// because we don't want to thrash here when counts are equal.
	sort.SliceStable(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	return counts[0].shardIdx
}

// PackageList contains the list of packages that a Shard can support
type PackageList []string

// Shard is a tuple of PackageList and ModelPath, representing the set of packages
// supported by the ExprModel at ModelPath.
type Shard struct {
	ModelPath string
	Packages  PackageList
}

// ShardedModel is a sharded expr model, sharded based on clusters of cooccuring packages
type ShardedModel struct {
	shards  []Shard
	options Options

	models      []Model
	lastUsed    []time.Time
	selectedIdx int32

	debug bool
}

// NewShardedModel creates a sharded model based on the shards provided
func NewShardedModel(ctx context.Context, shards []Shard, opts Options) (Model, error) {
	sm := &ShardedModel{
		shards:      shards,
		options:     opts,
		selectedIdx: 0,
	}

	for _, shard := range shards {
		model, err := newModelShard(shard, opts)
		if err != nil {
			return nil, err
		}
		sm.models = append(sm.models, model)
		sm.lastUsed = append(sm.lastUsed, time.Time{})
	}

	go sm.modelTimeout(ctx)

	return sm, nil
}

// Load will load all shards
func (m *ShardedModel) Load() error {
	for _, model := range m.models {
		model.Load()
	}
	return nil
}

// SelectShard will take the context of the current file to select which shard is the most
// most oppropriate to use based on a sharding/clustering criterea.
func (m *ShardedModel) SelectShard(ast *pythonast.Module) {
	imports := importsFromAST(ast)
	selectedIdx := selectShard(m.shards, imports)
	atomic.StoreInt32(&m.selectedIdx, selectedIdx)
	m.printf("SelectShard() called, selected shard %d", selectedIdx)
}

// --

// Reset unloads all shards
func (m *ShardedModel) Reset() {
	for _, shard := range m.models {
		shard.Reset()
	}
}

// IsLoaded returns true if the underlying model was successfully loaded.
func (m *ShardedModel) IsLoaded() bool {
	return m.models[atomic.LoadInt32(&m.selectedIdx)].IsLoaded()
}

// AttrSupported returns nil if the Model is able to provide completions for the
// specified parent.
func (m *ShardedModel) AttrSupported(rm pythonresource.Manager, parent pythonresource.Symbol) error {
	return m.models[atomic.LoadInt32(&m.selectedIdx)].AttrSupported(rm, parent)
}

// AttrCandidates for the specified parent symbol
func (m *ShardedModel) AttrCandidates(rm pythonresource.Manager, parent pythonresource.Symbol) ([]int32, []pythonresource.Symbol, error) {
	return m.models[atomic.LoadInt32(&m.selectedIdx)].AttrCandidates(rm, parent)
}

// CallSupported returns nil if the model is able to provide call completions for the
// specified symbol.
func (m *ShardedModel) CallSupported(rm pythonresource.Manager, sym pythonresource.Symbol) error {
	return m.models[atomic.LoadInt32(&m.selectedIdx)].CallSupported(rm, sym)
}

// Dir returns the directory from which the model was loaded.
func (m *ShardedModel) Dir() string {
	return m.models[atomic.LoadInt32(&m.selectedIdx)].Dir()
}

// FuncInfo gets all the needed info for call completion
func (m *ShardedModel) FuncInfo(rm pythonresource.Manager, sym pythonresource.Symbol) (*pythongraph.FuncInfo, error) {
	return m.models[atomic.LoadInt32(&m.selectedIdx)].FuncInfo(rm, sym)
}

// Predict an expression completion
func (m *ShardedModel) Predict(ctx kitectx.Context, in Input) (*GGNNResults, error) {
	imports := importsFromAST(in.RAST.Root)
	shard := selectShard(m.shards, imports)
	atomic.StoreInt32(&m.selectedIdx, shard)
	m.printf("Predict() called, selected shard %d", shard)

	m.lastUsed[shard] = time.Now()
	return m.models[shard].Predict(ctx, in)
}

// MetaInfo for the model
func (m *ShardedModel) MetaInfo() MetaInfo {
	return m.models[atomic.LoadInt32(&m.selectedIdx)].MetaInfo()
}

func (m *ShardedModel) printf(msg string, objs ...interface{}) {
	if m.debug {
		log.Printf("!!! [sharded_model] "+msg, objs...)
	}
}

func (m *ShardedModel) modelTimeout(ctx context.Context) {
	timer := time.NewTicker(checkModelsDuration)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			m.checkForIdleModels()
		}
	}
}

func (m *ShardedModel) checkForIdleModels() {
	defer func() {
		if r := recover(); r != nil {
			rollbar.PanicRecovery(r)
		}
	}()

	m.printf("checking if any shard is idle...")

	var loadedShards int
	for idx, lastUsed := range m.lastUsed {
		if (lastUsed == time.Time{}) {
			m.printf("shard %d not loaded", idx)
			continue
		}

		loadedShards++
		sinceLastUsed := time.Since(lastUsed)
		if time.Since(lastUsed) > modelUsedTimeout {
			m.printf("resetting shard %d, last used %s", idx, sinceLastUsed)
			m.models[idx].Reset()
			m.lastUsed[idx] = time.Time{}
		} else {
			m.printf("keeping shard %d, last used %s", idx, sinceLastUsed)
		}
	}

	loadedExprShards.Set(int64(loadedShards))
}

// --

func importsFromAST(ast *pythonast.Module) map[string]bool {
	imports := make(map[string]bool)

	// This logic is the same as the logic used to extract imports during clustering of packages
	// done in local-pipelines/python-imports-per-file/extract-imports/main.go
	pythonast.Inspect(ast, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}

		switch node := node.(type) {
		case *pythonast.ImportNameStmt:
			if len(node.Names) > 0 && node.Names[0].External != nil && len(node.Names[0].External.Names) > 0 {
				imports[node.Names[0].External.Names[0].Ident.Literal] = true
			}
			return false
		case *pythonast.ImportFromStmt:
			if node.Package != nil && len(node.Package.Names) > 0 {
				imports[node.Package.Names[0].Ident.Literal] = true
			}
			return false
		}

		return true
	})

	return imports
}
