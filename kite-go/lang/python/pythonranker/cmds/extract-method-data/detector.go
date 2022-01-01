package main

import (
	"log"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonranker"
	"github.com/kiteco/kiteco/kite-golib/re"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const (
	entropyThreshold = 8
)

// ontology maps a selector name to a set of possible
// fully qualified names. For example,
// namespace["min"] will be mapped to
// numpy.core.fromnumeric.amin
// builtins.min
// numpy.matrixlib.defmatrix.matrix.min
// numpy.core.fromnumeric.amin
// numpy.ma.core.MaskedArray.min
// numpy.ma.core.MaskedRecords.min
// numpy.ma.core.min
// numpy.ma.core.MaskedConstant.min
// numpy.ma.core.mvoid.min
type ontology struct {
	namespace map[string]map[string]struct{}
}

func newOntology() *ontology {
	return &ontology{
		namespace: make(map[string]map[string]struct{}),
	}
}

func (o *ontology) add(key, value string) {
	var names map[string]struct{}
	var exists bool
	if names, exists = o.namespace[key]; !exists {
		names = make(map[string]struct{})
		o.namespace[key] = names
	}
	names[value] = struct{}{}
}

func (o *ontology) find(key string) []string {
	var values []string
	log.Println(key)
	if names, found := o.namespace[key]; found {
		for name := range names {
			values = append(values, name)
		}
	}
	return values
}

// detector detects packages, methods in a so post
type detector struct {
	ranker          *pythonranker.PackageRanker
	graph           *pythonimports.Graph
	synonyms        map[string][]string
	packageOntology map[string]*ontology
}

func newDetector(ranker *pythonranker.PackageRanker,
	graph *pythonimports.Graph, synonyms map[string][]string) *detector {
	det := &detector{
		ranker:   ranker,
		graph:    graph,
		synonyms: synonyms,
	}
	det.packageOntology = constructOntology(ranker.Candidates(), graph)
	return det
}

func constructOntology(packages []string, graph *pythonimports.Graph) map[string]*ontology {
	packageOntology := make(map[string]*ontology)
	for _, p := range packages {
		ontByIDs := make(map[string]map[int64][]string)
		err := graph.Walk(p, func(name string, node *pythonimports.Node) bool {
			if node.Classification != pythonimports.Module && node.Classification != pythonimports.Type {
				return true
			}
			for m, child := range node.Members {
				if child == nil {
					return true
				}
				kind := child.Classification
				if kind != pythonimports.Function && kind != pythonimports.Descriptor {
					continue
				}
				fullname := strings.Join([]string{name, m}, ".")
				idToNames, found := ontByIDs[m]
				if !found {
					idToNames = make(map[int64][]string)
					ontByIDs[m] = idToNames
				}
				idToNames[child.ID] = append(idToNames[child.ID], fullname)
			}
			return true
		})
		if err != nil {
			log.Println(err)
		}
		ont := newOntology()
		for key, idToNames := range ontByIDs {
			for _, names := range idToNames {
				ont.add(key, names[0])
			}
		}
		packageOntology[p] = ont
	}
	return packageOntology
}

// detectFuncs takes a package name and a code block to detect whether any method names
// in the package exist in the code strings.
func (d *detector) detectFuncs(p, content string) []string {
	candidates := d.findFuncCandidates(content)
	var detectedMethods []string
	for _, cand := range candidates {
		tokens := strings.Split(cand, ".")
		var methods []string
		// if it looks like a fully qualified name, find it in the graph.
		if tokens[0] == p {
			node, err := d.graph.Find(cand)
			if err != nil {
				log.Printf("error encountered when finding %s in graph: %v\n", cand, err)
			}
			if node != nil && !node.CanonicalName.Empty() {
				methods = []string{node.CanonicalName.String()}
			} else {
				if ontology, found := d.packageOntology[p]; found {
					methods = ontology.find(tokens[len(tokens)-1])
				}
			}
		} else {
			if ontology, found := d.packageOntology[p]; found {
				methods = ontology.find(tokens[len(tokens)-1])
			}
		}
		if len(methods) != 0 {
			detectedMethods = append(detectedMethods, methods...)
		}
	}
	return text.Uniquify(detectedMethods)
}

// searchGraph walks through the graph rooted at p and finds all nodes that
// have a member sel.
func (d *detector) searchGraph(p, sel string) []string {
	var methods []string
	err := d.graph.Walk(p, func(name string, node *pythonimports.Node) bool {
		if node.Classification == pythonimports.Module || node.Classification == pythonimports.Type {
			target, found := node.Members[sel]
			if !found || target == nil {
				return true
			}
			if target.Classification == pythonimports.Function || target.CanonicalName.Equals("descriptor") {
				if !target.CanonicalName.Empty() {
					methods = append(methods, target.CanonicalName.String())
				} else {
					canonicalName, err := d.graph.CanonicalName(name)
					if err != nil {
						methods = append(methods, strings.Join([]string{canonicalName, sel}, "."))
					}
				}
			}
		}
		return true
	})
	if err != nil {
		log.Println(err)
	}
	return methods
}

// findFuncCandidates returns a list of methods exist in content.
func (d *detector) findFuncCandidates(content string) []string {
	selectors := re.SelRegexp.FindAllStringSubmatchIndex(content, -1)
	funcs := re.FuncRegexp.FindAllStringSubmatchIndex(content, -1)

	var candidates []string
	for _, sel := range selectors {
		candidates = append(candidates, content[sel[0]:sel[1]])
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
			candidates = append(candidates, content[fn[2]:fn[3]])
		}
	}
	return candidates
}

func (d *detector) detectPython(tags []string) ([]string, bool) {
	// check if the post is for python
	var isPython bool
	var filteredTags []string
	for _, tag := range tags {
		if match("python", d.synonyms[tag]) {
			isPython = true
		} else {
			filteredTags = append(filteredTags, tag)
		}
	}
	return filteredTags, isPython
}

func (d *detector) detectPackages(tags []string, title string) []string {
	// check what packages this post refers to
	var detectedPackages []string
	for _, tag := range tags {
		for _, dp := range d.synonyms[tag] {
			detectedPackages = append(detectedPackages, dp)
		}
	}
	if len(detectedPackages) > 0 {
		return text.Uniquify(detectedPackages)
	}

	data := d.ranker.Rank(title)

	// Use entropy to decide whether the post refers to any packages.
	// Empirically, with entropy < 8, it usually means that the post
	// actually refers to a specific package, so we use 8 as the
	// threshold.
	var scores []float64
	for _, d := range data {
		scores = append(scores, d.Score)
	}

	if entropy(scores) < entropyThreshold {
		return []string{data[0].Name}
	}
	return nil
}
