package stackoverflow

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

const (
	defaultRoot = "s3://kite-data/var-kite-data/stackoverflow_error_index"
)

var (
	supportedLanguages = []lang.Language{lang.Golang, lang.Python}
)

// The errorKey type for the index from errors to stackoverflow posts
type errorKey struct {
	language lang.Language
	errorID  int
}

// errorIndex maps errors to post ids
type errorIndex map[errorKey][]int

// LookupPosts looks up post ids based on provided language and error id.
func (index errorIndex) LookupPosts(language lang.Language, errorID int) []int {
	return index[errorKey{language, errorID}]
}

// --

func newDefaultErrorIndex() (errorIndex, error) {
	return newErrorIndex(defaultRoot)
}

func newErrorIndex(baseDir string) (errorIndex, error) {
	index := make(errorIndex)
	for _, l := range supportedLanguages {
		path := fileutil.Join(baseDir, l.Name()+".json")
		f, err := awsutil.NewCachedS3Reader(path)
		if err != nil {
			return nil, fmt.Errorf("could not open %s: %s", path, err)
		}
		err = loadJSON(f, l, index)
		if err != nil {
			return nil, fmt.Errorf("failed to load: %s: %s", path, err)
		}
		log.Println("loaded error index for", l.Name())
	}
	log.Printf("error index contains %d error/language pairs\n", len(index))
	return index, nil
}

func loadJSON(r io.Reader, language lang.Language, index errorIndex) error {
	// Declare the json structure. Keep within function scope to avoid confusing things.
	var indexJSON struct {
		Items []struct {
			ErrorID int   `json:"error_id"`
			PostIds []int `json:"post_ids"`
		} `json:"items"`
	}

	decoder := json.NewDecoder(r)
	err := decoder.Decode(&indexJSON)
	if err != nil {
		return err
	}

	log.Printf("Loaded an error index of size %d", len(indexJSON.Items))
	for _, item := range indexJSON.Items {
		index[errorKey{language, item.ErrorID}] = item.PostIds
	}

	return nil
}

func loadText(r io.Reader, language lang.Language, index errorIndex) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		tokens := strings.Split(scanner.Text(), " ")
		if len(tokens) == 0 {
			continue
		}

		errorID, err := strconv.Atoi(tokens[0])
		if err != nil {
			log.Println("Error parsing stackoverflow error index: ", err)
			continue
		}

		for _, t := range tokens[1:] {
			postID, err := strconv.Atoi(t)
			if err != nil {
				log.Println("Error parsing stackoverflow error index: ", err)
				continue
			}
			k := errorKey{language, errorID}
			index[k] = append(index[k], postID)
		}
	}

	// Check error state
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Error reading stackoverflow error index: %s", err)
	}

	return nil
}
