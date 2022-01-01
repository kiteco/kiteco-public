package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	patternMap map[string][]*pattern
	clusterDB  *leveldb.DB
)

type pattern struct {
	Method             string  `json:"method"`
	CanonicalSignature string  `json:"canon_sig"`
	Percentage         float64 `json:"percentage"`
	SigHash            string  `json:"sighash"`
}

func loadCanonicalClusters(patternFile, clusterDir string) error {
	err := loadPatterns(patternFile)
	if err != nil {
		return err
	}
	err = openClustersDB(clusterDir)
	if err != nil {
		log.Printf("cannot open the cluster db: %v", err)
		return err
	}
	return nil
}

func loadPatterns(patternFile string) error {
	patternMap = make(map[string][]*pattern)
	in, err := os.Open(patternFile)
	if err != nil {
		log.Printf("cannot open patterns file: %v\n", err)
		return err
	}
	decomp, err := gzip.NewReader(in)
	if err != nil {
		log.Printf("cannot open gzip reader: %v\n", err)
		return err
	}

	dec := json.NewDecoder(decomp)
	for {
		var pat pattern
		err = dec.Decode(&pat)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("cannot decode the pattern file: %v\n", err)
			return err
		}
		patternMap[pat.Method] = append(patternMap[pat.Method], &pat)
	}

	log.Println("Loaded patterns for", len(patternMap), "methods")
	return nil
}

func openClustersDB(clusterDir string) error {
	var err error
	clusterDB, err = leveldb.OpenFile(clusterDir, nil)
	return err
}

func clusterForSigHash(hash string) (*cluster, error) {
	value, err := clusterDB.Get([]byte(hash), nil)
	if err != nil {
		log.Printf("execution of GET failed for %s: %v\n", hash, err)
		return nil, err
	}

	var snippets []*pythoncode.Snippet
	err = json.Unmarshal(value, &snippets)
	if err != nil {
		log.Printf("can't unmarshal snippets: %v\n", err)
		return nil, err
	}

	cluster := &cluster{}
	for _, snip := range snippets {
		localSnip := &snippet{
			Code:      webutils.ColorizeCode([]byte(snip.Code)),
			Statement: webutils.ColorizeCode([]byte(snip.Incantations[0].Code)),
		}
		cluster.Snippets = append(cluster.Snippets, localSnip)
	}

	return cluster, nil
}

func clustersForMethod(method string) []*cluster {
	patterns, exists := patternMap[method]
	if !exists {
		return nil
	}

	var total float64
	var clusters []*cluster
	for _, pat := range patterns {
		if total > 0.90 {
			break
		}
		cluster, err := clusterForSigHash(pat.SigHash)
		if err != nil {
			return clusters
		}
		cluster.Percentage = pat.Percentage
		cluster.Representative = &snippet{
			Statement: webutils.ColorizeCode([]byte(pat.CanonicalSignature)),
		}

		clusters = append(clusters, cluster)
		total += cluster.Percentage
	}

	return clusters
}
