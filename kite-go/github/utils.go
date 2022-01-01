package github

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

// newJSONDecoder returns a json decoder for file.
func newJSONDecoder(file string) *json.Decoder {
	f, err := os.Open(file)
	if err != nil {
		log.Fatalf("Can't load %s\n", file)
	}
	decoder := json.NewDecoder(f)
	return decoder
}

// LoadPackageStats loads the package stats. An example
// of a stat file can be found at: /var/kite/data/packages.stats.json
func LoadPackageStats(file string) []*Package {
	decoder := newJSONDecoder(file)
	var ps []*Package
	for {
		var p Package
		err := decoder.Decode(&p)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("Can't decode the package file")
			return nil
		}
		ps = append(ps, &p)
	}
	return ps
}
