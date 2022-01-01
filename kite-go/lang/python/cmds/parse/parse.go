package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func valueStr(v pythontype.Value) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("%v (kind=%s)", v, v.Kind())
}

func main() {
	var repeat int
	var opts pythonparser.Options
	var printast, printtime, failFast, printwords, resolve bool
	flag.BoolVar(&opts.Trace, "trace", false, "print parse tree")
	flag.BoolVar(&failFast, "failfast", false, "exit after the first error")
	flag.IntVar(&opts.MaxDepth, "maxdepth", 0, "maximum recursion depth for parser")
	flag.IntVar(&repeat, "repeat", 1, "parse the same source file repeatedly (for performance)")
	flag.BoolVar(&printast, "print", true, "print the AST")
	flag.BoolVar(&opts.Approximate, "approx", true, "use the approx parser")
	flag.BoolVar(&printtime, "time", true, "print the parse duration")
	flag.BoolVar(&printwords, "printwords", true, "print the words")
	flag.BoolVar(&resolve, "resolve", false, "resolve the ast and print it")
	flag.Parse()

	f := os.Stdin
	if flag.NArg() > 0 {
		path := flag.Arg(0)
		opts.ScanOptions.Label = path

		var err error
		f, err = os.Open(path)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		opts.ScanOptions.Label = "<stdin>"
	}

	opts.ErrorMode = pythonparser.Recover
	if failFast {
		opts.ErrorMode = pythonparser.FailFast
	}

	src, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalln(err)
	}

	begin := time.Now()
	var module *pythonast.Module
	var words []pythonscanner.Word
	for i := 0; i < repeat; i++ {
		words, err = pythonscanner.Lex(src, opts.ScanOptions)
		if err != nil {
			log.Fatalf("error lexing source: %v\n", err)
		}

		module, err = pythonparser.ParseWords(kitectx.Background(), src, words, opts)
		if (err != nil && !opts.Approximate) || module == nil {
			log.Fatalln(err)
		}
	}
	duration := time.Since(begin)

	if printwords || printast {
		fmt.Printf("Source:\n%s\n", string(src))
	}

	if printwords {
		fmt.Println("Words:")
		for _, word := range words {
			fmt.Printf("%d:%d %s '%s'\n", word.Begin, word.End, word.Token.String(), word.Literal)
		}
	}

	if printast && !resolve {
		fmt.Println("AST:")
		pythonast.PrintPositions(module, os.Stdout, "\t")
	}

	if printtime {
		fmt.Println("Parse took", duration)
	}

	if resolve {
		fail(datadeps.Enable())
		opts := pythonresource.DefaultLocalOptions
		opts.Dists = []keytypes.Distribution{}

		rm, errc := pythonresource.NewManager(opts)
		fail(<-errc)

		rast, err := pythonanalyzer.NewResolver(rm, pythonanalyzer.Options{
			Path: "/src.py",
		}).Resolve(module)

		fail(err)

		var b bytes.Buffer
		var depth int
		pythonast.Inspect(module, func(n pythonast.Node) bool {
			if pythonast.IsNil(n) {
				depth--
				return false
			}
			depth++

			if expr, ok := n.(pythonast.Expr); ok {
				val := rast.References[expr]
				fmt.Fprintf(&b, "%s%s -> %s\n", strings.Repeat("  ", depth), pythonast.String(n), valueStr(val))
			} else {
				fmt.Fprintf(&b, "%s%s\n", strings.Repeat("  ", depth), pythonast.String(n))
			}

			return true
		})

		fmt.Println(b.String())
	}
}
