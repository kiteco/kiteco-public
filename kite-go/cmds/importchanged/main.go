package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"path"
	"strings"

	"golang.org/x/tools/refactor/importgraph"
)

const (
	kiteco = "github.com/kiteco/kiteco"
)

func main() {
	var target string
	flag.StringVar(&target, "target", "", "target to check against")
	flag.Parse()

	// Read in changed files
	scanner := bufio.NewScanner(os.Stdin)
	var changedFiles []string
	for scanner.Scan() {
		changedFiles = append(changedFiles, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln(err)
	}

	// Build the import graph
	forward, _, _ := importgraph.Build(&build.Default)
	reachable := forward.Search(target)

	// Filter out only kiteco imports
	kitecoImports := make(map[string]bool)
	for pkg, imported := range reachable {
		if imported && strings.HasPrefix(pkg, kiteco) {
			kitecoImports[pkg] = imported
		}
	}

	// Check to see if any of the changed files are in kiteco imports for the target
	// NOTE: Assumes changedFiles paths are relative to kiteco.
	var changed bool
	for _, cf := range changedFiles {
		filename := path.Join(kiteco, cf)
		pkg := strings.TrimSuffix(path.Dir(filename), "/")
		if _, exists := kitecoImports[pkg]; exists {
			fmt.Println(pkg, "->", filename)
			changed = true
		}
	}

	if changed {
		os.Exit(1)
	}
}
