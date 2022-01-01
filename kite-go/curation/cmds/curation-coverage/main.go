package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"sort"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func main() {
	var output string
	flag.StringVar(&output, "output", "", "file to dump output")
	flag.Parse()

	curatedSnippets := fileutil.Join(pythoncuration.DefaultSearchOptions.CurationRoot, "curated-snippets.emr")
	snippets := loadCuratedSnippets(curatedSnippets)

	stats := fileutil.Join(pythoncode.DefaultPipelineRoot, "merge_package_stats", "output")
	packageStats := loadPackageStats(stats)

	covered := make(map[string]bool)
	for _, snippet := range snippets {
		if snippet.Snippet == nil {
			continue
		}
		for _, inc := range snippet.Snippet.Incantations {
			covered[inc.ExampleOf] = true
		}
	}

	var methods []*pythoncode.MethodStats
	for _, stat := range packageStats {
		for _, m := range stat.Methods {
			methods = append(methods, m)
		}
	}

	sort.Sort(sort.Reverse(pythoncode.MethodsByCount(methods)))

	type identCoverage struct {
		Identifier string `json:"identifier"`
		Count      int    `json:"count"`
		Covered    bool   `json:"covered"`
	}

	out, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	enc := json.NewEncoder(out)
	for _, m := range methods {
		err = enc.Encode(&identCoverage{
			Identifier: m.Ident,
			Count:      m.Count,
			Covered:    covered[m.Ident],
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func loadCuratedSnippets(path string) []*pythoncuration.Snippet {
	r, err := fileutil.NewCachedReader(path)
	if err != nil {
		log.Fatal(err)
	}

	var snippets []*pythoncuration.Snippet
	iter := awsutil.NewEMRIterator(r)
	for iter.Next() {
		var snippet pythoncuration.Snippet
		err = json.Unmarshal(iter.Value(), &snippet)
		if err != nil {
			log.Fatal(err)
		}

		snippets = append(snippets, &snippet)
	}

	if err := iter.Err(); err != nil {
		log.Fatal(err)
	}

	return snippets
}

func loadPackageStats(path string) []*pythoncode.PackageStats {
	f, err := awsutil.NewShardedFile(path)
	if err != nil {
		log.Fatalln("cannot open completions dataset:", err)
	}

	var m sync.Mutex
	var packageStats []*pythoncode.PackageStats
	err = awsutil.EMRIterateSharded(f, func(key string, value []byte) error {
		var stats pythoncode.PackageStats
		err := json.Unmarshal(value, &stats)
		if err != nil {
			return err
		}

		m.Lock()
		packageStats = append(packageStats, &stats)
		m.Unlock()

		return nil
	})

	if err != nil {
		log.Fatalln("error reading completions:", err)
	}

	return packageStats
}
