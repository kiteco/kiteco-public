package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/javascript/ast"
	"github.com/kiteco/kiteco/kite-go/lang/javascript/parser"
	"github.com/montanaflynn/stats"
)

func main() {
	args := struct {
		File    string `arg:"positional"`
		Repeat  uint64 `arg:"help:parse the same source file repeatedly (for performance)"`
		Print   bool   `arg:"help:print the AST"`
		Time    bool   `arg:"help:print the parse duration"`
		Profile string `arg:"help:filename to write cpu profile"`
	}{
		Repeat: 1,
		Print:  true,
		Time:   true,
	}
	arg.MustParse(&args)

	if args.Profile != "" {
		if !strings.HasSuffix(args.Profile, ".prof") {
			args.Profile = args.Profile + ".prof"
		}

		f, err := os.Create(args.Profile)
		if err != nil {
			log.Fatalln(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	src, err := ioutil.ReadFile(args.File)
	if err != nil {
		log.Fatalln(err)
	}

	opts := parser.DefaultOptions
	opts.ModuleName = filepath.Base(args.File)

	var times []float64

	var module *ast.Node

	for i := uint64(0); i < args.Repeat; i++ {
		begin := time.Now()
		module, err = parser.Parse(src, opts)
		if err != nil {
			log.Fatalln(err)
		}
		times = append(times, float64(time.Since(begin)))
	}

	if args.Print {
		ast.PrintPositions(module, os.Stdout, "  ")
	}

	if args.Print {
		fmt.Printf("Parse time:\n")
		f, _ := stats.Median(times)
		fmt.Printf("  Median: %v\n", time.Duration(f))
		f, _ = stats.Mean(times)
		fmt.Printf("  Mean: %v\n", time.Duration(f))
		f, _ = stats.StdDevS(times)
		fmt.Printf("  StdDev: %v\n", time.Duration(f))
		f, _ = stats.Min(times)
		fmt.Printf("  Min: %v\n", time.Duration(f))
		f, _ = stats.Max(times)
		fmt.Printf("  Max: %v\n", time.Duration(f))
	}
}
