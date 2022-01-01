package main

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

func root(dotted string) string {
	if pos := strings.Index(dotted, "."); pos != -1 {
		return dotted[:pos]
	}
	return dotted
}

func main() {
	var args struct {
		Output string   `arg:"required"`
		Inputs []string `arg:"positional"`
	}
	arg.MustParse(&args)

	w, err := serialization.NewEncoder(args.Output)
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()

	for _, path := range args.Inputs {
		pkg := pythonimports.Package{
			Name: filepath.Base(path),
		}
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		deps := make(map[string]struct{})
		for _, line := range strings.Split(string(buf), "\n") {
			if line != "" {
				deps[root(line)] = struct{}{}
			}
		}
		for dep := range deps {
			pkg.Dependencies = append(pkg.Dependencies, dep)
		}
		w.Encode(pkg)
	}
}
