package main

import (
	"encoding/json"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/ranking"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/re"
	"github.com/kiteco/kiteco/kite-golib/text"
)

var (
	funcRegexp = regexp.MustCompile(`([a-zA-Z0-9_]+)\(`)
)

type byScore []*trainingDatum

func (b byScore) Len() int           { return len(b) }
func (b byScore) Swap(i, j int)      { b[j], b[i] = b[i], b[j] }
func (b byScore) Less(i, j int) bool { return b[i].Score < b[j].Score }

// loadRankingDB returns a map from query text to a list of labels
// that have a relevance score larger than 0.
func loadRankingDB() map[string][]ranking.Label {
	rankingDB := curation.GormDB(envutil.MustGetenv("RANKING_DB_DRIVER"), envutil.MustGetenv("RANKING_DB_URI"))
	manager := ranking.NewQueryManager(rankingDB)

	// load all queries
	queries, err := manager.GetAllQueries()
	if err != nil {
		log.Fatal(err)
	}
	queryIDToText := make(map[uint64]string)
	for _, q := range queries {
		queryIDToText[q.ID] = q.Text
	}

	queryToLabels := make(map[string][]ranking.Label)
	// load all labels
	labels, err := manager.GetAllLabels()
	if err != nil {
		log.Fatal(err)
	}

	for _, l := range labels {
		if l.Rank == 0 {
			continue
		}
		if text, found := queryIDToText[l.QueryID]; found {
			queryToLabels[text] = append(queryToLabels[text], l)
		}
	}
	return queryToLabels
}

// loadAttributes loads snippet's attributes
func loadAttributes(path string) (map[int64]*pythoncuration.Snippet, map[int64][]pythoncuration.Attribute) {
	snippets := make(map[int64]*pythoncuration.Snippet)
	attributes := make(map[int64][]pythoncuration.Attribute)

	s3r, err := fileutil.NewCachedReader(path)
	if err != nil {
		log.Fatalf("error loading curated snippets from %s: %v\n", path, err)
	}

	r := awsutil.NewEMRReader(s3r)
	for {
		_, value, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var cs pythoncuration.AnalyzedSnippet
		err = json.Unmarshal(value, &cs)

		if err != nil {
			log.Fatal(err)
		}

		if _, exists := attributes[cs.Snippet.Curated.Snippet.SnapshotID]; !exists {
			snippets[cs.Snippet.Curated.Snippet.SnapshotID] = cs.Snippet
			attributes[cs.Snippet.Curated.Snippet.SnapshotID] = cs.Attributes
		}
	}
	return snippets, attributes
}

// overlap returns overlapping selector names in set A and set B.
// We assume that set A contains only the selector names.
func overlap(setA, setB []string) []string {
	seen := make(map[string]struct{})
	for _, a := range setA {
		seen[a] = struct{}{}
	}

	var overlapping []string
	for _, b := range setB {
		tokens := strings.Split(b, ".")
		sel := strings.ToLower(tokens[len(tokens)-1])
		if _, found := seen[sel]; found {
			overlapping = append(overlapping, sel)
		}
	}
	return overlapping
}

// findFuncCandidates returns a list of methods exist in content.
func findFuncCandidates(content string) []string {
	selectors := re.SelRegexp.FindAllStringSubmatchIndex(content, -1)
	funcs := funcRegexp.FindAllStringSubmatchIndex(content, -1)

	var candidates []string
	for _, sel := range selectors {
		candidates = append(candidates, strings.ToLower(content[sel[4]:sel[5]]))
	}
	for _, fn := range funcs {
		var contained bool
		for _, sel := range selectors {
			if fn[2] > sel[0] && fn[2] < sel[1] {
				contained = true
				break
			}
		}
		if !contained {
			candidates = append(candidates, strings.ToLower(content[fn[2]:fn[3]]))
		}
	}
	return candidates
}

// buildLookUpTable builds a look up table that maps from a selector
// name to its fully qualified names.
func buildLookUpTable(packages map[string]struct{}, attrs []pythoncuration.Attribute, methods []string) map[string][]string {
	lookup := make(map[string][]string)
	for _, att := range attrs {
		tokens := strings.Split(att.Type, ".")
		if _, found := packages[tokens[0]]; !found {
			continue
		}
		sel := tokens[len(tokens)-1]
		full := att.Type

		if att.Attribute != "" {
			sel = att.Attribute
			full = strings.Join([]string{att.Type, att.Attribute}, ".")
		}
		sel = strings.ToLower(sel)
		lookup[sel] = append(lookup[sel], full)
	}
	for _, m := range methods {
		tokens := strings.Split(m, ".")
		if len(tokens) > 0 {
			sel := tokens[len(tokens)-1]
			sel = strings.ToLower(sel)
			lookup[sel] = append(lookup[sel], m)
		}
	}
	return lookup
}

// findCanonicalNames finds the canonical names for rawNames.
func findCanonicalNames(graph *pythonimports.Graph, rawNames []string) []string {
	var names []string
	for _, ident := range text.Uniquify(rawNames) {
		_, err := graph.CanonicalName(ident)
		if err == nil {
			names = append(names, ident)
			continue
		}
		tokens := strings.Split(ident, ".")
		sel := tokens[len(tokens)-1]
		p := tokens[0]

		var candidates []string
		err = graph.Walk(p, func(name string, node *pythonimports.Node) bool {
			node, found := node.Members[sel]
			if found && node != nil {
				cand := node.CanonicalName.String()
				if cand != "" {
					cand = strings.Join([]string{name, sel}, ".")
					if !strings.Contains(cand, "._") {
						candidates = append(candidates, cand)
					}
				}
			}
			return true
		})
		if err != nil {
			log.Fatal(err)
		}
		for _, cand := range text.Uniquify(candidates) {
			_, err := graph.Find(cand)
			if err != nil {
				log.Println(err)
				continue
			}
			if strings.Split(cand, ".")[0] == p {
				names = append(names, cand)
			}
		}
	}
	return names
}
