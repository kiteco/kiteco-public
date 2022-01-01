package pythoncuration

import (
	"io/ioutil"
	"log"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
	yaml "gopkg.in/yaml.v1"
)

// CollectionToPackages represents the mapping between the collection name used
// on the curation platform and canonical package names.
// For example, for the collection for `builtin-types`, the relevant package
// names will be `__builtin__`.
type CollectionToPackages struct {
	Collection       string   `yaml:"collection"`
	RelevantPackages []string `yaml:"relevant_packages"`
}

func loadCollectionToPackages(path string) map[string]*CollectionToPackages {
	in, err := fileutil.NewCachedReader(path)
	if err != nil {
		log.Fatal(err)
	}

	var collectionToPackages []*CollectionToPackages
	bytes, _ := ioutil.ReadAll(in)
	err = yaml.Unmarshal(bytes, &collectionToPackages)
	if err != nil {
		log.Fatal(err)
	}

	rules := make(map[string]*CollectionToPackages)
	for _, c := range collectionToPackages {
		rules[c.Collection] = c
	}
	return rules
}
