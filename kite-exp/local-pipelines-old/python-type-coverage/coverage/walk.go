package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	largeFileThreshold = int(1e5) // files over this size (in bytes) may be noise
	itersDataProp      = 3
)

type batchStats struct {
	// Corpus is the name of the corpus
	Corpus string
	// Files is the number of python files
	Files int64
	// Toolarge is the number of python files that were too large
	TooLarge int64
	// ParseErrors is the number of python files that had parse errors
	ParseErrors int64
	// Added is the number of python files that were added to the batch
	Added int64
	// ProcessingTime is the duration spent processing the batch
	ProcessingTime time.Duration
	// Trace is the trace from data flwo analysis and capabilities inference
	Trace string
}

type collector struct {
	exprs map[pythonast.Expr]pythontype.Value
}

// MissingName implements pythonstatic.PropagatorDelegate
func (c *collector) MissingName(*pythonast.NameExpr) {}

// MissingAttr implements pythonstatic.PropagatorDelegate
func (c *collector) MissingAttr(*pythonast.AttributeExpr, pythontype.Value) {}

// MissingCall implements pythonstatic.PropagatorDelegate
func (c *collector) MissingCall(*pythonast.CallExpr) {}

// Resolved implements pythonstatic.PropagatorDelegate
func (c *collector) Resolved(expr pythonast.Expr, val pythontype.Value) {
	c.exprs[expr] = val
}

type visitor func(files map[string]sourceFile, collector *collector, stats batchStats) error

type sourceFile struct {
	AST      *pythonast.Module
	Contents []byte
	Path     string
}

type astFetcher map[string]sourceFile

func (a astFetcher) FetchAST(path string) *pythonast.Module {
	return a[path].AST
}

type walkParams struct {
	Corpus       string
	LibraryDepth int
}

func walk(params walkParams, visitor visitor) error {
	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		return fmt.Errorf("error loading import graph: %v", err)
	}
	if err := pythonskeletons.UpdateGraph(graph); err != nil {
		return fmt.Errorf("error updating graph with skeletons: %v", err)
	}

	typeinducer, err := typeinduction.LoadModel(graph, typeinduction.DefaultClientOptions)
	if err != nil {
		return fmt.Errorf("error loading typeinduction client: %v", err)
	}
	pythonskeletons.UpdateReturnTypes(graph, typeinducer)

	dirs, err := ioutil.ReadDir(params.Corpus)
	if err != nil {
		return fmt.Errorf("error reading dir %s: %v", params.Corpus, err)
	}

	opts := pythonstatic.DefaultOptions
	opts.UseCapabilities = true
	opts.LibraryDepth = params.LibraryDepth
	for _, di := range dirs {
		if !di.IsDir() {
			continue
		}

		start := time.Now()
		files := make(map[string]sourceFile)
		libs := make(astFetcher)
		stats := batchStats{Corpus: filepath.Join(params.Corpus, di.Name())}
		if err := filepath.Walk(stats.Corpus, func(path string, fi os.FileInfo, err error) error {
			switch {
			case err != nil:
				return err
			case !strings.HasSuffix(path, ".py"):
				return nil
			}

			isLib := pythonbatch.IsLibrary(path)
			if !isLib {
				stats.Files++
			}

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("error reading file `%s`: %v", path, err)
			}

			if len(content) > largeFileThreshold {
				if !isLib {
					stats.TooLarge++
				}
				return nil
			}

			mod, err := pythonparser.Parse(kitectx.Background(), content, pythonparser.Options{
				ErrorMode: pythonparser.FailFast,
				ScanOptions: pythonscanner.Options{
					Label: path,
				},
			})

			if err != nil {
				if !isLib {
					stats.ParseErrors++
				}
				return nil
			}

			abs, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("unable to make absolute path for `%s`: %v\n", path, err)
			}

			sf := sourceFile{
				Path:     path,
				AST:      mod,
				Contents: content,
			}

			if isLib {
				libs[abs] = sf
			} else {
				files[abs] = sf
				stats.Added++
			}

			return nil
		}); err != nil {
			return fmt.Errorf("failed to walk directory `%s`: %v", params.Corpus, err)
		}

		var libFiles []string
		for f := range libs {
			libFiles = append(libFiles, f)
		}
		lsm := pythonstatic.NewLibraryManager(libFiles, libs)

		assembler := pythonstatic.NewAssembler(graph, typeinducer, lsm, opts)
		var trace bytes.Buffer
		assembler.SetTrace(&trace)
		for f, sf := range files {
			assembler.AddSource(pythonstatic.ASTBundle{AST: sf.AST, Path: f, Imports: pythonstatic.FindImports(sf.AST)})
		}

		collector := &collector{
			exprs: make(map[pythonast.Expr]pythontype.Value),
		}
		assembler.Build(collector)
		stats.ProcessingTime = time.Since(start)
		stats.Trace = trace.String()

		if err := visitor(files, collector, stats); err != nil {
			return fmt.Errorf("visitor reported error on corpus `%s`: %v", params.Corpus, err)
		}
	}
	return nil
}
