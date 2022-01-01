package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"log"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func isAllowed(counts map[string]int) bool {
	if val, exists := counts["repository"]; exists {
		if val > 10 {
			return true
		}
	}
	return false
}

func loadAllowedIdentifiers(groupedFile string, targets map[string]struct{}) map[string]map[string]struct{} {
	allowed := make(map[string]map[string]struct{})
	in, err := fileutil.NewCachedReader(groupedFile)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	r := awsutil.NewEMRIterator(in)
	for r.Next() {
		var grouped pythoncode.GroupedStats
		err := json.Unmarshal(r.Value(), &grouped)
		if err != nil {
			log.Fatal(err)
		}

		if _, isTarget := targets[grouped.Package]; isTarget && isAllowed(grouped.Counts) {
			allowedIdents, exists := allowed[grouped.Package]
			if !exists {
				allowedIdents = make(map[string]struct{})
				allowed[grouped.Package] = allowedIdents
			}
			allowedIdents[grouped.Identifier] = struct{}{}
		}
	}

	if err := r.Err(); err != nil {
		log.Fatal(err)
	}

	return allowed
}

// --

type statsByCount []*pythoncode.PackageStats

func (s statsByCount) Len() int           { return len(s) }
func (s statsByCount) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s statsByCount) Less(i, j int) bool { return s[i].Count < s[j].Count }

func loadPackageStats(statsFile string, allowed map[string]map[string]struct{}) []*pythoncode.PackageStats {
	var pkgStats []*pythoncode.PackageStats

	in, err := fileutil.NewCachedReader(statsFile)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	r := awsutil.NewEMRIterator(in)
	for r.Next() {
		var stats pythoncode.PackageStats
		err := json.Unmarshal(r.Value(), &stats)
		if err != nil {
			log.Fatal(err)
		}
		if allowedIdents, isAllowed := allowed[stats.Package]; isAllowed {
			var count int
			var methods []*pythoncode.MethodStats
			for _, m := range stats.Methods {
				if _, exists := allowedIdents[m.Ident]; exists {
					methods = append(methods, m)
					count += m.Count
				}
			}
			stats.Count = count
			stats.Methods = methods
			pkgStats = append(pkgStats, &stats)
		}
	}

	if err := r.Err(); err != nil {
		log.Fatal(err)
	}

	sort.Sort(sort.Reverse(statsByCount(pkgStats)))
	return pkgStats
}

func loadDocumentation(path string) pythondocs.Modules {
	in, err := fileutil.NewCachedReader(path)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	decomp, err := gzip.NewReader(in)
	if err != nil {
		log.Fatal(err)
	}

	dec := gob.NewDecoder(decomp)
	modules := make(pythondocs.Modules)
	err = modules.DecodeGob(dec)
	if err != nil {
		log.Fatal(err)
	}

	return modules
}

func loadTargets(path string) map[string]struct{} {
	if path == "" {
		return nil
	}
	targets := make(map[string]struct{})
	buf, err := Asset(path)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(buf))
	for scanner.Scan() {
		targets[scanner.Text()] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return targets
}

func loadBuiltinDocstrings(graph *pythonimports.Graph, graphstrings pythonimports.GraphStrings) map[string]string {
	identifiers := make(map[string]string)
	for b := range python.Builtins {
		node, err := graph.Find(python.BuiltinPackage + "." + b)
		if err != nil {
			continue
		}
		for k, child := range node.Members {
			if child == nil {
				continue
			}
			if strings.HasPrefix(k, "__") && strings.HasSuffix(k, "__") {
				continue
			}
			if member, ok := graphstrings[child.ID]; ok {
				identifiers[python.BuiltinPackage+"."+b+"."+k] = member.Docstring
			}
		}
		if s, ok := graphstrings[node.ID]; ok {
			identifiers[python.BuiltinPackage+"."+b] = s.Docstring
		}
	}
	return identifiers
}
