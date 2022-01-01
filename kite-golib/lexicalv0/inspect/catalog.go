package inspect

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"hash/crc64"
	"io/ioutil"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
)

const prefix = "s3://kite-local-pipelines/lexical-inspector/catalog"

const generation = 1

// generation 1:
//   - addresses the addition of the Sample.Query.Language and Sample.Generation fields.
//   - for generation 0 samples, we set Sample.Query.Language to Golang.

// Save saves a sample and returns a key for reloading the sample.
func Save(sample Sample) (string, error) {
	serialized, err := json.Marshal(sample)
	if err != nil {
		return "", err
	}
	key, _ := Key(sample)
	path := fileutil.Join(prefix, key)
	writer, err := fileutil.NewBufferedWriter(path)
	defer writer.Close()
	if err != nil {
		return "", err
	}
	zipWriter := gzip.NewWriter(writer)
	defer zipWriter.Close()
	_, err = zipWriter.Write(serialized)
	if err != nil {
		return "", err
	}
	return key, nil
}

// Load returns the sample saved with the given key.
func Load(key string) (Sample, error) {
	path := fileutil.Join(prefix, key)
	reader, err := fileutil.NewReader(path)
	if err != nil {
		return Sample{}, err
	}
	defer reader.Close()
	zipReader, err := gzip.NewReader(reader)
	defer zipReader.Close()
	if err != nil {
		return Sample{}, err
	}
	contents, err := ioutil.ReadAll(zipReader)
	if err != nil {
		return Sample{}, err
	}
	var sample Sample
	err = json.Unmarshal(contents, &sample)
	if err != nil {
		return Sample{}, err
	}
	if sample.Generation == 0 {
		sample.Query.Language = lexicalv0.NewLangGroup(lang.Golang)
	}
	return sample, nil
}

// ListKeys returns a list of keys for all saved samples.
func ListKeys() ([]string, error) {
	paths, err := fileutil.ListDir(prefix)
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, path := range paths {
		pieces := strings.Split(path, "/")
		keys = append(keys, pieces[len(pieces)-1])
	}
	return keys, nil
}

// Key hashes a sample
func Key(sample Sample) (string, error) {
	serialized, err := json.Marshal(sample)
	if err != nil {
		return "", err
	}
	table := crc64.MakeTable(crc64.ECMA)
	return fmt.Sprintf("%d", crc64.Checksum(serialized, table)), nil
}
