package pythonexpr

import (
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

func newTestShardedModel(shards []Shard, opts Options) *ShardedModel {
	sm := &ShardedModel{
		shards:      shards,
		options:     opts,
		selectedIdx: 0,
	}

	for _, shard := range shards {
		model, err := newMockModel(shard.ModelPath, opts)
		if err != nil {
			panic("newMockModel shouldn't return an error")
		}
		sm.models = append(sm.models, model)
		sm.lastUsed = append(sm.lastUsed, time.Time{})
	}

	return sm
}

func Test_ShardedModel(t *testing.T) {
	shards := []Shard{
		{
			ModelPath: "s3://path/to/a/model/shard0",
			Packages:  []string{"foo1", "foo2", "foo3"},
		},
		{
			ModelPath: "s3://path/to/a/model/shard1",
			Packages:  []string{"bar1", "bar2", "bar3"},
		},
	}

	model := newTestShardedModel(shards, DefaultOptions)

	require.Equal(t, len(shards), len(model.shards))
	require.Equal(t, len(shards), len(model.models))

	file1 := `
import foo1
import foo2
import bar1
`

	file2 := `
import bar1
import bar2
import foo1
`

	file3 := `
import bar1
import bar2
import foo1
import foo2
`

	file1AST := requireAST(t, file1)
	file2AST := requireAST(t, file2)
	file3AST := requireAST(t, file3)

	require.Equal(t, int32(0), model.selectedIdx)

	// Select model shard 0
	model.SelectShard(file1AST)
	require.Equal(t, int32(0), model.selectedIdx)
	require.Equal(t, shards[0].ModelPath, model.Dir())

	// Select model shard 1
	model.SelectShard(file2AST)
	require.Equal(t, int32(1), model.selectedIdx)
	require.Equal(t, shards[1].ModelPath, model.Dir())

	// Select model shard 0
	model.SelectShard(file1AST)
	require.Equal(t, int32(0), model.selectedIdx)
	require.Equal(t, shards[0].ModelPath, model.Dir())

	// Select model shard 1
	model.SelectShard(file2AST)
	require.Equal(t, int32(1), model.selectedIdx)
	require.Equal(t, shards[1].ModelPath, model.Dir())

	// Test model selection stability when either shard would work
	model.SelectShard(file3AST)
	selected := model.selectedIdx

	// Try it a bunch of times
	for i := 0; i < 10; i++ {
		model.SelectShard(file3AST)
		require.Equal(t, selected, model.selectedIdx)
		require.Equal(t, shards[selected].ModelPath, model.Dir())
	}
}

func Test_ShardSelection_Single(t *testing.T) {
	shards := []Shard{
		{
			ModelPath: "s3://path/to/a/model/shard0",
			Packages:  []string{"import-a-1", "import-a-2", "import-a-3"},
		},
	}

	type testCase struct {
		imports          []string
		expectedModelIdx int32
	}

	testCases := []testCase{
		{imports: []string{"import-a-1", "import-a-2", "import-b-1"}, expectedModelIdx: 0},
		{imports: []string{"import-b-1", "import-b-2", "import-a-1"}, expectedModelIdx: 0},
		{imports: []string{"rando-1", "rando-2", "rando-3"}, expectedModelIdx: 0},
		{imports: []string{}, expectedModelIdx: 0},
		{imports: nil, expectedModelIdx: 0},
	}

	for _, tc := range testCases {
		selectedIdx := selectShard(shards, toMap(tc.imports))
		require.Equal(t, tc.expectedModelIdx, selectedIdx)
	}
}

func Test_ShardSelection_Multi(t *testing.T) {
	shards := []Shard{
		{
			ModelPath: "s3://path/to/a/model/shard0",
			Packages:  []string{"import-a-1", "import-a-2", "import-a-3"},
		},
		{
			ModelPath: "s3://path/to/a/model/shard1",
			Packages:  []string{"import-b-1", "import-b-2", "import-b-3"},
		},
	}

	type testCase struct {
		imports          []string
		expectedModelIdx int32
	}

	testCases := []testCase{
		{imports: []string{"import-a-1", "import-a-2", "import-b-1"}, expectedModelIdx: 0},
		{imports: []string{"import-b-1", "import-b-2", "import-a-1"}, expectedModelIdx: 1},
		{imports: []string{}, expectedModelIdx: 0},
		{imports: nil, expectedModelIdx: 0},
	}

	for _, tc := range testCases {
		selectedIdx := selectShard(shards, toMap(tc.imports))
		require.Equal(t, tc.expectedModelIdx, selectedIdx)
	}
}

func Test_Predict(t *testing.T) {
	shards := []Shard{
		{
			ModelPath: "s3://path/to/a/model/shard0",
			Packages:  []string{"foo1", "foo2", "foo3"},
		},
		{
			ModelPath: "s3://path/to/a/model/shard1",
			Packages:  []string{"bar1", "bar2", "bar3"},
		},
	}

	file1 := `
import foo1
import foo2
import bar1
`

	file2 := `
import bar1
import bar2
import foo1
`
	model := newTestShardedModel(shards, DefaultOptions)

	model.Predict(kitectx.Background(), Input{RAST: requireRAST(t, file1)})
	require.Equal(t, model.models[0].(*mockModel).getCalledCount("Predict"), 1)
	require.Equal(t, model.models[1].(*mockModel).getCalledCount("Predict"), 0)

	model.Predict(kitectx.Background(), Input{RAST: requireRAST(t, file2)})
	require.Equal(t, model.models[0].(*mockModel).getCalledCount("Predict"), 1)
	require.Equal(t, model.models[1].(*mockModel).getCalledCount("Predict"), 1)

	model.Predict(kitectx.Background(), Input{RAST: requireRAST(t, file1)})
	require.Equal(t, model.models[0].(*mockModel).getCalledCount("Predict"), 2)
	require.Equal(t, model.models[1].(*mockModel).getCalledCount("Predict"), 1)
}

// --

func requireAST(t *testing.T, buf string) *pythonast.Module {
	words, err := pythonscanner.Lex([]byte(buf), pythonscanner.DefaultOptions)
	require.NoError(t, err)

	var ast *pythonast.Module
	err = kitectx.Background().WithTimeout(time.Second, func(ctx kitectx.Context) error {
		var err error
		ast, err = pythonparser.ParseWords(ctx, []byte(buf), words, pythonparser.Options{})
		return err
	})
	require.Nil(t, err)
	require.NotNil(t, ast)
	return ast
}

func requireRAST(t *testing.T, buf string) *pythonanalyzer.ResolvedAST {
	return &pythonanalyzer.ResolvedAST{
		Root: requireAST(t, buf),
	}
}

func toMap(vals []string) map[string]bool {
	m := make(map[string]bool)
	for _, val := range vals {
		m[val] = true
	}
	return m
}
