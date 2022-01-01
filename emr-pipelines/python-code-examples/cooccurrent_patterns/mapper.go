package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	funcRegexp = regexp.MustCompile(`\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
)

type snippetCooccur struct {
	Functions []string
	Hash      string
	Score     int
}

func main() {
	r := awsutil.NewEMRReader(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	for {
		_, value, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var snippet pythoncode.Snippet
		err = json.Unmarshal(value, &snippet)
		if err != nil {
			log.Fatal(err)
		}

		functions := parseFunctions(snippet.Code)
		hash := snippet.Hash().String()
		score := len(snippet.Code)

		cooccur := snippetCooccur{
			Functions: functions,
			Hash:      hash,
			Score:     score,
		}

		buf, err := json.Marshal(cooccur)
		if err != nil {
			log.Fatal(err)
		}

		seen := make(map[string]struct{})
		for _, inc := range snippet.Incantations {
			if _, exists := seen[inc.ExampleOf]; !exists {
				seen[inc.ExampleOf] = struct{}{}
				if found(inc.ExampleOf, functions) {
					err = w.Emit(inc.ExampleOf, buf)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
}

func found(s string, patterns []string) bool {
	tokens := strings.Split(s, ".")
	target := tokens[len(tokens)-1]
	for _, p := range patterns {
		if target == p {
			return true
		}
	}
	return false
}

func parseFunctions(code string) []string {
	var functions []string
	seen := make(map[string]struct{})
	matches := funcRegexp.FindAllStringSubmatch(code, -1)
	for _, m := range matches {
		if _, exists := seen[m[1]]; !exists {
			functions = append(functions, m[1])
			seen[m[1]] = struct{}{}
		}
	}
	return functions
}
