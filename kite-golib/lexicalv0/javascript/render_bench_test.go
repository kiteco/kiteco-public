package javascript

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/javascript"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer/treesitter"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/render"
)

// Benchmarks for the FormatCompletion function - which includes parsing the
// source with treesitter and rendering a pretty-printed version of the source
// by calling Prettify.
//
// Some external benchmarks were also done by running the ../cmds/test_prettify_js
// command in a batch on a data set of multiple popular javascript repositories
// from github. This is just to get a high-level idea of performance as the command
// does a few things:
//
// - for each repository (there are 15 of them), the go command is rebuilt
//   (that's because the test calls the diff.sh bash script for each repo)
// - on each js file, the source is parsed twice (once to get the AST of the
//   original source, and once to parse the prettified output to compare its
//   resulting AST with the original's)
// - both ASTs are then walked and compared
//
// So there's more going on than just calling Prettify. That being said, the
// results on a Macbook Pro 2015 8GB RAM with SSD drive are:
//
// * 3242 js files processed in 7m30s (7.2 files per second)
// * ~35000 KB (~34 MB) processed (78KB per second)
//
// Update:
// * 6m39 (8.1 files per second)
// * 87.5KB per second

func BenchmarkFormatCompletion(b *testing.B) {
	printMemoryUsage(b)
	b.Run("Tiny", benchmarkFormatCompletionTiny)
	b.Run("Small", benchmarkFormatCompletionSmall)
	b.Run("Medium", benchmarkFormatCompletionMedium)
	b.Run("Large", benchmarkFormatCompletionLarge)

	time.Sleep(time.Second)
	runtime.GC()
	printMemoryUsage(b)
}

func BenchmarkTreeSitterParseAndWalkNoop(b *testing.B) {
	b.Run("Tiny", func(b *testing.B) {
		benchmarkTreeSitterParseAndWalkNoop(b, srcFromTestdataFile(b, "tiny.js"))
	})
	b.Run("Small", func(b *testing.B) {
		benchmarkTreeSitterParseAndWalkNoop(b, srcFromTestdataFile(b, "small.js"))
	})
	b.Run("Medium", func(b *testing.B) {
		benchmarkTreeSitterParseAndWalkNoop(b, srcFromTestdataFile(b, "medium.js"))
	})
	b.Run("Large", func(b *testing.B) {
		benchmarkTreeSitterParseAndWalkNoop(b, srcFromTestdataFile(b, "large.js"))
	})
}

func benchmarkFormatCompletionTiny(b *testing.B) {
	snip := `append()`
	benchmarkFormatCompletion(b, srcFromTestdataFile(b, "tiny.js"), snip)
}

func benchmarkFormatCompletionSmall(b *testing.B) {
	snip := `[node(), ascii()]`
	benchmarkFormatCompletion(b, srcFromTestdataFile(b, "small.js"), snip)
}

func benchmarkFormatCompletionMedium(b *testing.B) {
	snip := `throw new ERR_SYSTEM_ERROR(ctx);`
	benchmarkFormatCompletion(b, srcFromTestdataFile(b, "medium.js"), snip)
}

func benchmarkFormatCompletionLarge(b *testing.B) {
	snip := `[ jqXHR, statusText, error ]`
	benchmarkFormatCompletion(b, srcFromTestdataFile(b, "large.js"), snip)
}

var (
	ResultComp   data.Snippet
	ResultStr    string
	ResultUint32 uint32
	ResultSym    sitter.Symbol
	ResultPt     sitter.Point
	ResultNode   *sitter.Node
)

func benchmarkFormatCompletion(b *testing.B, src, snippet string) {
	replIx := strings.Index(src, "$$")
	src = strings.Replace(src, "$$", "", -1)
	compl := data.Completion{
		Snippet: data.Snippet{
			Text: snippet,
		},
		Replace: data.Selection{
			Begin: replIx,
			End:   replIx,
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// assign to exported package-level var to prevent compiler optimizations
		// from messing with the benchmark.
		ResultComp = FormatCompletion(src, compl, DefaultPrettifyConfig, render.MatchEnd)
	}
}

func benchmarkTreeSitterParseAndWalkNoop(b *testing.B, src string) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		func() {
			parser := sitter.NewParser()
			defer parser.Close()
			parser.SetLanguage(javascript.GetLanguage())
			tree := parser.Parse([]byte(src))
			defer tree.Close()
			treesitter.Inspect(tree.RootNode(), func(n *sitter.Node) bool {
				if n != nil {
					ResultStr = n.Type()
					ResultUint32 = n.ChildCount()
					ResultSym = n.Symbol()
					ResultUint32 = n.StartByte()
					ResultUint32 = n.EndByte()
					ResultPt = n.StartPoint()
					ResultPt = n.EndPoint()
					ResultNode = n.Parent()
				}
				return true
			})
		}()
	}
}

func srcFromTestdataFile(b *testing.B, file string) string {
	src, err := ioutil.ReadFile(filepath.Join("testdata", file))
	if err != nil {
		b.Fatal(err)
	}
	return string(src)
}

func printMemoryUsage(b *testing.B) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	b.Log(">>>> ", m.Alloc, m.TotalAlloc)
}
