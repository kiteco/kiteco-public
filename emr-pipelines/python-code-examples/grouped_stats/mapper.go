package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

func byFunction(s *pythoncode.Snippet) string {
	return s.Hash().String()
}

func byFile(s *pythoncode.Snippet) string {
	return s.FromFile
}

func byDir(s *pythoncode.Snippet) string {
	return filepath.Dir(s.FromFile)
}

func byRepository(s *pythoncode.Snippet) string {
	// The first two path components contain the name of the repository from which the code example came
	parts := strings.Split(s.FromFile, string(filepath.Separator))
	if len(parts) < 2 {
		return ""
	}
	return filepath.Join(parts[0], parts[1])
}

// usage represents a package that was used in a particular aggregation pool,
// which could be a file, directory, or repository.
type usage struct {
	Package    string
	Identifier string
	Group      string
	Aggregate  string
}

func main() {
	groupers := map[string]func(*pythoncode.Snippet) string{
		"function":   byFunction,
		"file":       byFile,
		"dir":        byDir,
		"repository": byRepository,
	}

	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	seen := make(map[usage]struct{})

	for r.Next() {
		var snippet pythoncode.Snippet
		err := json.Unmarshal(r.Value(), &snippet)
		if err != nil {
			log.Fatal(err)
		}

		for aggregate, grouper := range groupers {
			group := grouper(&snippet)
			if group == "" {
				continue
			}

			// Emit the method name of every incantation, keyed by the
			// the root module name of the method.
			for _, attr := range snippet.Attributes {
				parts := strings.Split(attr, ".")
				if len(parts) == 0 {
					continue
				}

				// Emit usage for package
				p1 := usage{
					Package:    parts[0],
					Identifier: parts[0],
					Group:      group,
					Aggregate:  aggregate,
				}

				// Emit usage for full identifier
				p2 := usage{
					Package:    parts[0],
					Identifier: attr,
					Group:      group,
					Aggregate:  aggregate,
				}

				for _, p := range []usage{p1, p2} {
					if _, duplicate := seen[p]; !duplicate {
						buf, err := json.Marshal(p)
						if err != nil {
							log.Fatalln(err)
						}

						w.Emit(p.Identifier, buf)
						seen[p] = struct{}{}
					}
				}
			}
		}
	}

	if err := r.Err(); err != nil {
		log.Fatal(err)
	}
}
