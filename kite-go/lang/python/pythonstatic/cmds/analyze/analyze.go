package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

func parseFiles(dir string) map[string]*pythonast.Module {
	asts := make(map[string]*pythonast.Module)

	// Parse each file
	var files, tooLarge, parseErrs, added int64
	if err := filepath.Walk(dir, func(srcpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || !strings.HasSuffix(srcpath, ".py") {
			return nil
		}
		files++

		if info.Size() > 1000000 {
			tooLarge++
			return nil
		}

		buf, err := ioutil.ReadFile(srcpath)
		if err != nil {
			return err
		}

		ast, err := pythonparser.Parse(kitectx.Background(), buf, pythonparser.Options{})
		if err != nil {
			parseErrs++
			log.Println(err)
			return nil
		}

		added++
		asts[srcpath] = ast
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Corpus %s contained %d python files, %d were too large, %d contained parse errors, %d were processed.\n",
		dir, files, tooLarge, parseErrs, added)

	return asts
}

type byName []*pythontype.Symbol

func (xs byName) Len() int           { return len(xs) }
func (xs byName) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }
func (xs byName) Less(i, j int) bool { return xs[i].Name.String() < xs[j].Name.String() }

func main() {
	var args struct {
		Input   string `arg:"positional,required"`
		NumRuns int
		UseCaps bool `arg:"help:Use capabilities during data flow analysis"`
	}
	args.NumRuns = 1
	arg.MustParse(&args)

	// TODO if we use a custom import graph, this is broken
	manager, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		log.Fatal(err)
	}

	if !path.IsAbs(args.Input) {
		abs, err := filepath.Abs(args.Input)
		if err != nil {
			log.Fatal(err)
		}
		args.Input = abs
	}

	asts := parseFiles(args.Input)

	opts := pythonstatic.DefaultOptions
	opts.UseCapabilities = args.UseCaps
	opts.AllowValueMutation = true

	var assembly *pythonstatic.Assembly
	var trace bytes.Buffer
	var durations []time.Duration
	for i := 0; i < args.NumRuns; i++ {
		ai := pythonstatic.AssemblerInputs{
			Graph: manager,
		}
		assembler := pythonstatic.NewAssembler(kitectx.Background(), ai, opts)
		for path, mod := range asts {
			assembler.AddSource(pythonstatic.ASTBundle{AST: mod, Path: path, Imports: pythonstatic.FindImports(kitectx.Background(), path, mod)})
		}

		if i == args.NumRuns-1 {
			assembler.SetTrace(&trace)
		}

		start := time.Now()
		var err error
		assembly, err = assembler.Build(kitectx.Background())
		if err != nil {
			log.Fatalln(err)
		}
		d := time.Since(start)
		fmt.Printf("Took %v to build batch\n", d)
		durations = append(durations, d)
	}

	fmt.Println(trace.String())

	var srcs, libs []string
	symsByFile := make(map[string][]*pythontype.Symbol)
	assembly.WalkSymbols(func(sym *pythontype.Symbol) {
		if _, seen := symsByFile[sym.Name.File]; !seen {
			if _, found := asts[sym.Name.File]; found {
				srcs = append(srcs, sym.Name.File)
			} else {
				libs = append(libs, sym.Name.File)
			}
		}
		symsByFile[sym.Name.File] = append(symsByFile[sym.Name.File], sym)
	})

	sort.Strings(srcs)
	sort.Strings(libs)

	for _, srcpaths := range [][]string{libs, srcs} {
		for _, srcpath := range srcpaths {
			if srcpath == "builtins" {
				continue
			}
			fmt.Println("\n", srcpath)

			syms := symsByFile[srcpath]
			sort.Sort(byName(syms))

			for _, sym := range syms {
				tail := sym.Name.Path.Last()
				if strings.HasPrefix(tail, "__") && strings.HasSuffix(tail, "__") {
					continue
				}
				log.Printf("  %-40v := %v", sym.Name.Path, sym.Value)
			}
		}
	}

	var avg float64
	for _, d := range durations {
		avg += float64(d)
	}
	avg *= 1. / float64(len(durations))
	fmt.Println("Average batch build time:", time.Duration(avg))
}
