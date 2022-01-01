package main

import (
	"encoding/json"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/performancetest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

func main() {
	log.SetPrefix("")
	log.SetFlags(0)

	if len(os.Args) != 1 && len(os.Args) != 3 || len(os.Args) == 3 && os.Args[1] != "--json" {
		fmt.Println("usage: performancetest [--json output.json]")
		os.Exit(1)
	}

	jsonOutputFile := ""
	if len(os.Args) == 3 {
		jsonOutputFile = os.Args[2]
	}

	if err := datadeps.Enable(); err != nil {
		log.Fatal(err)
	}

	mgr, errC := pythonresource.NewManager(pythonresource.DefaultLocalOptions)
	if err := <-errC; err != nil {
		log.Fatalf("error creating resource manager: %v", err)
	}

	allCompletions := durationHistogram{}
	totalTime := durationHistogram{}
	firstInvocation := durationHistogram{}

	// iterate files in tests/*.py, then providers for each file
	var allStats performancetest.ProviderStatsList

	for _, testFilePath := range locateTestData() {
		if statList, err := performancetest.TestProviders(mgr, testFilePath); err != nil {
			log.Fatalln(err)
		} else {
			allStats = append(allStats, statList...)
		}
	}

	// one dataset per provider for the current file
	for _, stats := range allStats {
		totalTime.add(stats.TotalDuration())

		if stats.Empty() {
			continue
		}

		firstInvocation.add(stats.Durations()[0])
		for _, d := range stats.Durations() {
			allCompletions.add(d)
		}
	}

	// print to file, if set on the cmdline
	if jsonOutputFile != "" {
		jsonBytes, err := json.Marshal(allStats)
		if err != nil {
			log.Fatalf("error marshalling json: %s", err.Error())
		}

		if err := ioutil.WriteFile(jsonOutputFile, jsonBytes, 0600); err != nil {
			log.Fatalf("unable to create json file: %s", err.Error())
		}
	}

	fmt.Println()
	fmt.Printf("Processing time for one new completion item (all providers):\n%s", allCompletions.String())

	fmt.Println()
	fmt.Printf("Total processing time (per provider and file):\n%s", totalTime.String())

	fmt.Println()
	fmt.Printf("Time to first completion (per provider and file):\n%s", firstInvocation.String())
}

func locateTestData() []string {
	// try to locate relative to this cmd and based on $GOPATH
	// GOPATH is the fallback for a setup where the binary is not stored in this cmd directory
	testDir, _ := filepath.Abs(filepath.Join("..", "..", "performancetest", "tests"))
	if _, err := os.Stat(testDir); err != nil {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			gopath = build.Default.GOPATH
		}
		testDir, _ = filepath.Abs(filepath.Join(gopath,
			"src", "github.com", "kiteco", "kiteco", "kite-go",
			"lang", "python", "pythoncomplete", "performancetest", "tests"))

		if _, err := os.Stat(testDir); err != nil {
			log.Fatalln(err)
		}
	}

	paths, err := filepath.Glob(filepath.Join(testDir, "*.py"))
	if err != nil {
		log.Fatalln(err)
	}
	sort.Strings(paths)
	return paths
}
