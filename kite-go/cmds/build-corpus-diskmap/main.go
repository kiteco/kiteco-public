package main

import (
	"flag"
	"log"
	"reflect"
	"strconv"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/diskmap"
)

func main() {
	var outputPath string
	flag.StringVar(&outputPath, "output", "./pythondocstrings.diskmap", "where to write the JSON diskmap")
	flag.Parse()

	graph, graphErr := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if graphErr != nil {
		log.Fatalln("unable to load graph from " + pythonimports.DefaultImportGraph)
	}

	oldPackageStats := "s3://kite-emr/users/tarak/python-code-examples/2016-01-21_15-47-59-PM/merge_package_stats/output"
	pkgStats, pkgErr := pythoncode.LoadGithubPackageStats(oldPackageStats)
	if pkgErr != nil {
		log.Fatalln("Could not load the package stats")
	}
	gh := pythoncode.NewGithubPrior(graph, pkgStats)

	corpus, corpusErr := pythondocs.LoadEntities(graph, pythondocs.DefaultSearchOptions)

	if corpusErr != nil {
		log.Fatalln("error initializing entities")
	}

	log.Println("loaded", len(corpus.Entities), "entities")

	builder := diskmap.NewBuilder()
	var dmError error
	for node, entity := range corpus.Entities {
		key := strconv.FormatInt(node.ID, 10)
		entity.Full = entity.FullIdent() // unfortunately, this getter mutates the object
		entity.Score = gh.Find(entity.Full)
		dmError = diskmap.JSON.Add(builder, key, entity)
		if dmError != nil {
			log.Fatalln("couldn't serialize")
		}
	}

	writeErr := builder.WriteToFile(outputPath)
	if writeErr != nil {
		log.Fatalln("failed to write to disk")
	}

	getter, openErr := diskmap.NewMap(outputPath)
	if openErr != nil {
		log.Fatalln("could not open")
	}

	for node, entity := range corpus.Entities {
		key := strconv.FormatInt(node.ID, 10)
		var obj pythondocs.LangEntity
		jsonErr := diskmap.JSON.Get(getter, key, &obj)

		if jsonErr != nil {
			log.Fatalln("could not deserialize the object")
		}

		if !reflect.DeepEqual(&obj, entity) {
			log.Fatalf("object was not equal")
		}
	}
}
