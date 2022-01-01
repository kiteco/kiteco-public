package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

func main() {
	flag.Parse()

	f := os.Stdin
	if flag.NArg() > 0 {
		var err error
		path := flag.Arg(0)
		f, err = os.Open(path)
		if err != nil {
			log.Fatalln(err)
		}
	}

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	file := pythonscanner.File(buf)
	lines := strings.Split(string(buf), "\n")

	var prevline int
	var opts pythonscanner.Options
	scanner := pythonscanner.NewScanner(buf, opts)
	for {
		begin, end, tok, lit := scanner.Scan()
		line := file.Line(begin)
		var linestr string
		if line > prevline {
			linestr = strings.TrimSpace(lines[line-1])
		}
		prevline = line

		fmt.Printf("%8d...%8d %-12s %-40s | %s\n", begin, end, tok.String(), lit, linestr)
		if tok == pythonscanner.EOF {
			break
		}
	}
}
