package main

import (
	"encoding/gob"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/ranking"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	_ "github.com/mattn/go-sqlite3"
)

const (
	logPrefix = "[passive-features] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

var (
	defaultCurationPath = "s3://kite-emr/datasets/curated-snippets/2015-10-29_10-10-12-AM/"
)

func main() {
	var (
		test         string
		dir          string
		curationRoot string
	)
	flag.StringVar(&test, "test", "", "test queries")
	flag.StringVar(&dir, "dir", "", "dir for the output files")
	flag.StringVar(&curationRoot, "curation", defaultCurationPath, "path to the curated snippets (.emr)")
	flag.Parse()

	if test == "" || dir == "" {
		flag.Usage()
		log.Fatal("must specifiy --test --dir")
	}

	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)

	// connect to the ranking label database
	rankingDB := curation.GormDB(
		envutil.MustGetenv("RANKING_DB_DRIVER"),
		envutil.MustGetenv("RANKING_DB_URI"))

	rankingManager := ranking.NewQueryManager(rankingDB)

	// load curated snippets
	curationPath := fileutil.Join(curationRoot, "curated-snippets.emr")
	snippets := loadSnippets(curationPath)

	// instantiate featurer
	featurer := pythoncuration.NewExampleFeaturer()

	// load test queries
	testQueries := loadTestQueries(test)

	// select queries for passive search
	queries, err := rankingManager.SelectQueryByType(ranking.Passive)
	if err != nil {
		log.Fatal("can't get passive queries", err)
	}
	var trainEntries []ranking.Entry
	var testEntries []ranking.Entry

	var numTest int
	var numTrain int

	for _, q := range queries {
		// skip any queries that contain space, which should not happen.
		// need to investiaget and see when those queries got inserted to
		// the database.
		if strings.Contains(q.Text, " ") {
			continue
		}

		labels, err := rankingManager.SelectLabelsByQueryID(q.ID)
		if err != nil {
			log.Fatalln("can't find labels for query id", q.ID, err)
		}

		// is none of the code examples are relevant, skip this sample set
		var hasRelevant bool
		for _, l := range labels {
			if l.Rank > 0 {
				hasRelevant = true
			}
		}
		if !hasRelevant {
			continue
		}

		log.Println(q.Text)

		_, isTest := testQueries[q.Text]
		if isTest {
			numTest++
		} else {
			numTrain++
		}

		for _, l := range labels {
			snippet, exists := snippets[l.SnapshotID]
			if !exists {
				log.Println("cannot find snapshot id", l.SnapshotID, "query is:", q.Text)
				continue
			}

			entry := ranking.Entry{
				SnapshotID: l.SnapshotID,
				Label:      l.Rank,
				QueryHash:  q.Hash(),
				QueryText:  q.Text,
				Features:   featurer.Features(q.Text, snippet, nil),
			}

			if isTest {
				testEntries = append(testEntries, entry)
			} else {
				trainEntries = append(trainEntries, entry)
			}
		}
	}
	log.Printf("Loaded %d training queries", numTrain)
	log.Printf("Loaded %d test  queries", numTest)

	testPayload := map[string]interface{}{
		"FeatureLabels":   featurer.Labels(),
		"Data":            testEntries,
		"FeaturerOptions": nil,
	}

	trainPayload := map[string]interface{}{
		"FeatureLabels":   featurer.Labels(),
		"Data":            trainEntries,
		"FeaturerOptions": nil,
	}

	// Encode training data
	ftrainJSON, err := os.Create(path.Join(dir, "train.json"))
	if err != nil {
		log.Fatal(err)
	}
	defer ftrainJSON.Close()

	w := json.NewEncoder(ftrainJSON)
	err = w.Encode(trainPayload)
	if err != nil {
		log.Fatal(err)
	}

	// Encode test data
	ftestJSON, err := os.Create(path.Join(dir, "test.json"))
	if err != nil {
		log.Fatal(err)
	}
	defer ftestJSON.Close()

	w = json.NewEncoder(ftestJSON)
	err = w.Encode(testPayload)
	if err != nil {
		log.Fatal(err)
	}

	// save the featurers
	ffeat, err := os.Create(path.Join(dir, "featurer.gob"))
	if err != nil {
		log.Fatal(err)
	}
	defer ffeat.Close()

	encoder := gob.NewEncoder(ffeat)
	err = encoder.Encode(featurer)
	if err != nil {
		log.Fatal(err)
	}
}
