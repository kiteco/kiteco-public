package main

import (
	"bufio"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

var importAliases = map[string][]string{
	"matplotlib.pyplot": {"plt"},
	"numpy":             {"np"},
	"tensorflow":        {"tf"},
	"pandas":            {"pd"},
	"seaborn":           {"sns"},
}

var aliasCount = make(map[string]int)

func main() {
	log.SetPrefix("")
	log.SetFlags(0)

	var err error

	if len(os.Args) < 2 {
		log.Fatalln("usage: spyder-data <output-file> [max-depth, default: 2]")
	}
	targetFile := os.Args[1]

	maxDepth := 2
	if len(os.Args) == 3 {
		if maxDepth, err = strconv.Atoi(os.Args[2]); err != nil {
			log.Fatalln(err)
		}
	}

	if err = datadeps.Enable(); err != nil {
		log.Fatal(err)
	}

	opts := pythonresource.DefaultOptions
	opts.Manifest = opts.Manifest.Filter("SymbolGraph", "ReturnTypes") // names from pythonresource/internal/resources/resources.go

	mgr, errc := pythonresource.NewManager(opts)
	if err := <-errc; err != nil {
		log.Fatalf("could not load resource manager: %s", err)
	}

	childPaths := make(map[string]bool, 100000)
	collectFromRoots(mgr, maxDepth, childPaths)

	paths := flatten(childPaths)
	sort.Strings(paths)
	writeToDisk(targetFile, paths)

	log.Println()
	log.Println("Aliases:")
	for prefix, count := range aliasCount {
		log.Printf("%s: %d", prefix, count)
	}
}

// collects child elements of dist up to maxDepth
// root is at depth 1, the first child below a root is at depth 2, etc.
func collectFromRoots(mgr pythonresource.Manager, maxDepth int, childPaths map[string]bool) {
	for _, dist := range mgr.Distributions() {
		if dist == keytypes.BuiltinDistribution2 {
			log.Printf("\tskipping dist %s", dist.String())
			continue
		}
		log.Printf("dist: %s\n", dist.Name)

		topLevels, err := mgr.TopLevels(dist)
		if err != nil {
			log.Fatalln(err)
		}

		for _, name := range topLevels {
			log.Printf("\titerating root symbol %s", name)
			root, err := mgr.NewSymbol(dist, pythonimports.NewPath(name))
			if err != nil {
				log.Fatalln(err)
			}

			err = collectChildPaths(mgr, root, maxDepth, childPaths)
			if err != nil {
				log.Fatalf("error iterating %s in dist %s: %s", name, dist.Name, err.Error())
			}
		}
	}
}

// collectChildPaths recursively collects child symbols from the given parent symbol
// it iterates elements breadth-first
// it returns without collecting symbols when the depth at invocation is larger than maxDepth
func collectChildPaths(mgr pythonresource.Manager, parent pythonresource.Symbol, maxDepth int, target map[string]bool) error {
	parentPath := parent.Path()

	depth := len(parentPath.Parts)
	for prefix := range importAliases {
		if parentPath.HasPrefix(prefix) {
			depth -= strings.Count(prefix, ".")
			break
		}
	}
	if depth >= maxDepth {
		return nil
	}

	childNames, err := mgr.Children(parent)
	if err != nil {
		return errors.Errorf("error retrieving children for %s: %s", parent.PathString(), err.Error())
	}

	var next []pythonresource.Symbol
	for _, childName := range childNames {
		childSymbol, err := mgr.ChildSymbol(parent, childName)
		if err != nil {
			// fixme: skip or return?
			log.Printf("error retrieving child symbol %s for parent %s: %s\n", childName, parent.PathString(), err.Error())
			continue
		}

		// only add new functions which are not private and have known return types
		if !target[childSymbol.PathString()] &&
			mgr.Kind(childSymbol) == keytypes.FunctionKind &&
			len(mgr.ReturnTypes(childSymbol)) >= 0 &&
			!strings.HasPrefix(childName, "_") {

			path := childSymbol.Path()
			pathStr := path.String()

			target[pathStr] = true

			// add aliases, if a prefix is contained in the global alias mapping
			for prefix, aliases := range importAliases {
				if path.HasPrefix(prefix) {
					for _, alias := range aliases {
						aliased := alias + strings.TrimPrefix(pathStr, prefix)
						target[aliased] = true
						aliasCount[prefix]++
					}
				}
			}

			// fixme don't add to next for now
			continue
		}

		// e.g. when an error occurred...
		// register to iterate for depth+1
		next = append(next, childSymbol)
	}

	for _, n := range next {
		err = collectChildPaths(mgr, n, maxDepth, target)
		if err != nil {
			return err
		}
	}

	return nil
}

func flatten(childPaths map[string]bool) []string {
	var paths []string
	for path, _ := range childPaths {
		paths = append(paths, path)
	}
	return paths
}

func writeToDisk(targetFile string, paths []string) {
	f, err := os.OpenFile(targetFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalln(err)
	}
	buffer := bufio.NewWriter(f)
	defer f.Close()
	defer buffer.Flush()
	for _, p := range paths {
		_, err = buffer.WriteString(p)
		if err != nil {
			log.Fatalln(err)
		}

		_, err = buffer.Write([]byte{'\n'})
		if err != nil {
			log.Fatalln(err)
		}
	}
}
