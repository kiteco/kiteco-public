package main

// TODO:
//  - split string literals when necessary
//  - include comments in output

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/codewrap"
	"github.com/kiteco/kiteco/kite-go/lang"
)

func main() {
	var tabWidth int
	var columns int
	flag.IntVar(&columns, "cols", 45, "Number of columns in which to format code")
	flag.IntVar(&tabWidth, "tabwidth", 2, "Number of spaces per tab in output")
	flag.Parse()

	opts := codewrap.Options{
		Columns:  columns,
		TabWidth: tabWidth,
	}

	for i := 0; i < flag.NArg(); i++ {
		path := flag.Arg(i)
		err := filepath.Walk(path, func(srcpath string, srcinfo os.FileInfo, err error) error {
			if err != nil {
				return err
			} else if srcinfo.IsDir() || !strings.HasSuffix(srcpath, lang.Golang.Extension()) {
				return nil
			}

			rel, _ := filepath.Rel(path, srcpath)
			fmt.Println("\nProcessing " + rel)

			// Read file into buffer
			buf, err := ioutil.ReadFile(srcpath)
			if err != nil {
				return err
			}

			// Format the code
			tokens, _ := codewrap.TokenizeGolang(buf)
			flow := codewrap.Layout(tokens, opts)

			// Print results
			codewrap.PrintSideBySide(flow, opts)
			return nil
		})
		if err != nil {
			fmt.Println(err)
		}
	}
}
