package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

// loadRawPackageStats loads the raw package stats we gather from github.
func loadRawPackageStats(path string) map[string]pythoncode.PackageStats {
	f, err := awsutil.NewShardedFile(path)
	if err != nil {
		log.Fatalln("cannot open completions dataset:", err)
	}
	packageData := make(map[string]pythoncode.PackageStats)
	var m sync.Mutex
	err = awsutil.EMRIterateSharded(f, func(key string, value []byte) error {
		var stats pythoncode.PackageStats
		err := json.Unmarshal(value, &stats)
		if err != nil {
			return err
		}
		m.Lock()
		packageData[stats.Package] = stats
		m.Unlock()
		return nil
	})
	if err != nil {
		log.Fatalln("error reading completions:", err)
	}
	return packageData
}

// githubStats loads and adjusts the function stats we gathered from github hierarchically.
// We don't have very good stats on class method usage, and this function is a hack to get that.
// It first computes the method distribution within a class p(m|p) (gathered by hierarchyCount),
// and then adjust the class method count by doing p(m|p) * count(p). Count(p) is stored
// in flatCount.
func githubStats(path string, prior map[string]map[string]float64, parsedModules parsedModules) {
	packageStats := loadRawPackageStats(path)
	for p, stats := range packageStats {
		if _, exists := prior[p]; !exists {
			continue
		}
		parsedModule := parsedModules.find(p)
		if parsedModule == nil {
			continue
		}
		for _, m := range stats.Methods {
			entity := parsedModule.findEntity(m.Ident)
			if entity != nil {
				entity.addCount(m.Count)
			}
		}
		for m := range prior[p] {
			prior[p][m] = parsedModule.estimateCount(m)
		}
	}
}
