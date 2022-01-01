package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func loadSnippets(path string) map[int64]*pythoncuration.Snippet {
	snippetMap := make(map[int64]*pythoncuration.Snippet)

	s3r, err := fileutil.NewCachedReader(path)
	if err != nil {
		log.Fatalf("error loading curated snippets from %s: %v\n", path, err)
	}
	defer s3r.Close()

	r := awsutil.NewEMRIterator(s3r)
	for r.Next() {
		var cs pythoncuration.Snippet
		err = json.Unmarshal(r.Value(), &cs)
		if err != nil {
			log.Fatal(err)
		}
		snippetMap[cs.Curated.Snippet.SnapshotID] = &cs
	}
	if err := r.Err(); err != nil {
		log.Printf("error reading %s: %v\n", path, err)
	}
	return snippetMap
}

func loadTestQueries(path string) map[string]struct{} {
	// load test queries
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	queries := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		queries[scanner.Text()] = struct{}{}
	}
	return queries
}
