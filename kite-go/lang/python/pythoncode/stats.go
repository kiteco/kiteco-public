package pythoncode

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const (
	minLogPrior = -30
)

// GithubPrior computes the prior popularity of entities in an import graph based on
// github stats.
type GithubPrior struct {
	stats map[string]*PackagePrior
}

// NewGithubPrior returns a pointer to a new github prior object.
func NewGithubPrior(graph *pythonimports.Graph, packageStats map[string]PackageStats) *GithubPrior {
	stats := make(map[string]*PackagePrior)

	var total int
	for pkg, data := range packageStats {
		pkg = strings.ToLower(pkg)
		prior, err := NewPackagePrior(graph, data)
		if err != nil {
			log.Printf("can't build entity prior for %s: %s\n", pkg, err.Error())
			continue
		}

		stats[pkg] = prior
		total += data.Count
	}

	logTotal := math.Log(float64(total))
	// Set prior for the pacakges
	for pkg, data := range packageStats {
		pkg = strings.ToLower(pkg)
		if entityPrior, exists := stats[pkg]; exists {
			entityPrior.SetRootLogProb(math.Log(float64(data.Count)) - logTotal)
		}
	}

	return &GithubPrior{
		stats: stats,
	}
}

// Find returns the prior probability of the given identifier.
func (p *GithubPrior) Find(ident string) float64 {
	pkg := strings.ToLower(strings.Split(ident, ".")[0])

	if entityPrior, found := p.stats[pkg]; found {
		return entityPrior.RootLogProb() + entityPrior.ChainedLogProb(ident)
	}
	return minLogPrior * 2
}

// PackagePrior computes the prior probability of each entity in a package.
type PackagePrior struct {
	nameToNode  map[string]*Node
	idToNames   map[int64][]string
	root        *Node
	rootLogProb float64
}

// NewPackagePriorFromUniqueNameCounts takes in a identifier to count map and builds a prior
func NewPackagePriorFromUniqueNameCounts(pkg string, identCounts map[string]int) (*PackagePrior, error) {
	prior := &PackagePrior{
		nameToNode: make(map[string]*Node),
		idToNames:  make(map[int64][]string),
	}

	// set up the root
	prior.nameToNode[pkg] = &Node{
		name: pkg,
	}
	id := int64(len(prior.idToNames))
	prior.idToNames[id] = append(prior.idToNames[id], pkg)
	prior.root = prior.nameToNode[pkg]

	// go through the methods and build the chart
	for m, count := range identCounts {
		id := int64(len(prior.idToNames))
		prior.idToNames[id] = append(prior.idToNames[id], m)
		err := prior.insert(m, count)
		if err != nil {
			return nil, err
		}
	}
	for id, names := range prior.idToNames {
		prior.idToNames[id] = text.Uniquify(names)
	}
	backwardPropogate(prior.root)
	forwardNormalize(prior.root)
	return prior, nil

}

// NewPackagePrior takes in an import graph and the raw package stats to build
// a PackagePrior object.
func NewPackagePrior(graph *pythonimports.Graph, rawStats PackageStats) (*PackagePrior, error) {
	n, err := graph.Find(rawStats.Package)
	if err != nil {
		return nil, err
	}

	prior := &PackagePrior{
		nameToNode: make(map[string]*Node),
		idToNames:  make(map[int64][]string),
	}

	prior.nameToNode[rawStats.Package] = &Node{
		name: rawStats.Package,
	}
	prior.idToNames[n.ID] = append(prior.idToNames[n.ID], rawStats.Package)

	prior.root = prior.nameToNode[rawStats.Package]

	// go through the methods and build the chart
	for _, m := range rawStats.Methods {
		n, err := graph.Find(m.Ident)
		if err != nil {
			continue
		}
		prior.idToNames[n.ID] = append(prior.idToNames[n.ID], m.Ident)
		err = prior.insert(m.Ident, m.Count)
		if err != nil {
			return nil, err
		}

	}
	for id, names := range prior.idToNames {
		prior.idToNames[id] = text.Uniquify(names)
	}
	backwardPropogate(prior.root)
	forwardNormalize(prior.root)
	return prior, nil
}

// SetRootLogProb sets the prior for the root node of an entity prior.
func (p *PackagePrior) SetRootLogProb(log float64) {
	p.rootLogProb = log
}

// RootLogProb returns the log probability of the package that the entity prior corresponds to.
func (p *PackagePrior) RootLogProb() float64 {
	return p.rootLogProb
}

// ChainedLogProb returns the chained log probability of the given identifier. If the identifier
// can't be found, then return minLogPrior.
func (p *PackagePrior) ChainedLogProb(ident string) float64 {
	node, found := p.nameToNode[ident]
	if !found {
		return minLogPrior
	}
	return node.chainedlogProb
}

// insert inserts a node with the given name and count
func (p *PackagePrior) insert(name string, count int) error {
	// find node for this identifier
	node, found := p.nameToNode[name]
	if !found {
		node = &Node{
			name: name,
		}
		p.nameToNode[name] = node
		// attach this node to its parent
		parent, err := p.findParent(name)
		if err != nil {
			return err
		}
		parent.children = append(parent.children, node)
	}
	node.count += count
	return nil
}

// EntityLogProbs returns a map from an entity's name to its log prior probability.
func (p *PackagePrior) EntityLogProbs() map[string]float64 {
	logProbs := make(map[string]float64)
	for _, names := range p.idToNames {
		var probs []float64
		for _, name := range names {
			probs = append(probs, p.nameToNode[name].logProb)
		}
		logProbs[names[0]] = logSumExp(probs) - math.Log(float64(len(probs)))
	}
	return logProbs
}

// EntityChainedLogProbs returns a map from an entity's name to its chained log prior probability.
func (p *PackagePrior) EntityChainedLogProbs() map[string]float64 {
	logProbs := make(map[string]float64)
	for _, names := range p.idToNames {
		var probs []float64
		for _, name := range names {
			probs = append(probs, p.nameToNode[name].chainedlogProb)
		}
		logProbs[names[0]] = logSumExp(probs)
	}
	return logProbs
}

// findParent find the parent of a node. If the parent doesn't exist, it creates
// the parent node along with all the missing grand parent node.
func (p *PackagePrior) findParent(name string) (*Node, error) {
	parts := strings.Split(name, ".")
	parentName := strings.Join(parts[:len(parts)-1], ".")

	if parentName == "" {
		// heuristically try possible package names
		parentNode, found := p.nameToNode[strings.ToLower(parts[0])]
		if !found {
			return nil, fmt.Errorf("parentName is empty. Should've found the root. got name: %s", name)
		}
		return parentNode, nil
	}

	parentNode, found := p.nameToNode[parentName]
	if !found {
		parentNode = &Node{
			name: parentName,
		}
		p.nameToNode[parentName] = parentNode
		grandParent, err := p.findParent(parentName)
		if err != nil {
			return nil, err
		}
		grandParent.children = append(grandParent.children, parentNode)
	}
	return parentNode, nil
}

// Node represents an entity in a package.
type Node struct {
	name           string
	children       []*Node
	count          int
	chainedlogProb float64
	logProb        float64
}

// --

// forwardNormalize computes the log probability of each node in the graph.
func forwardNormalize(node *Node) {
	var total int
	for _, c := range node.children {
		total += c.count
	}
	if total == 0 && len(node.children) > 0 {
		panic("sum of children counts should not be zero")
	}
	for _, c := range node.children {
		c.chainedlogProb = node.chainedlogProb + math.Log(float64(c.count)/float64(total))
		c.logProb = math.Log(float64(c.count) / float64(total))
		forwardNormalize(c)
	}
}

// backwardPropogate propagate children counts to the parent if the count for the
// parent is 0.
func backwardPropogate(node *Node) {
	var total int
	for _, c := range node.children {
		backwardPropogate(c)
		total += c.count
	}
	if node.count == 0 {
		node.count = total
	}
}

// logSumExp receives a slice of log scores: log(a), log(b), log(c)...
// and returns log(a + b + c....)
func logSumExp(logs []float64) float64 {
	var max float64
	for _, l := range logs {
		if l > max {
			max = l
		}
	}
	var sum float64
	for _, l := range logs {
		sum += math.Exp(l - max)
	}
	return max + math.Log(sum)
}

// --

// LoadGithubPackageStats loads the raw package stats we gather from github.
func LoadGithubPackageStats(path string) (map[string]PackageStats, error) {
	f, err := awsutil.NewShardedFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open github package stats %s: %v", path, err)
	}
	packageData := make(map[string]PackageStats)
	var m sync.Mutex
	err = awsutil.EMRIterateSharded(f, func(key string, value []byte) error {
		var stats PackageStats
		err := json.Unmarshal(value, &stats)
		if err != nil {
			return err
		}
		m.Lock()
		packageData[stats.Package] = stats
		m.Unlock()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error reading package stats file: %v", err)
	}
	return packageData, nil
}

// LoadGithubCooccurenceStats loads the raw github co occurence stats we gather from github.
// Returns map from (top level) package/module name to a map containing the (top level) packages/modules that occured
// in the same file as the given package along with the counts.
func LoadGithubCooccurenceStats(path string) (map[string]map[string]int64, error) {
	pkgs := make(map[string]map[string]int64)
	f, err := fileutil.NewCachedReader(path)
	if err != nil {
		return nil, fmt.Errorf("error opening cooccurence stats %s: %v", path, err)
	}

	r := awsutil.NewEMRIterator(f)

	for r.Next() {
		var cooccurs map[string]int64
		if err := json.Unmarshal(r.Value(), &cooccurs); err != nil {
			return nil, fmt.Errorf("error unmarshalling cooccurences for package %s: %v", r.Key(), err)
		}

		pkgs[r.Key()] = cooccurs
	}

	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("error reading cooccurences from %s: %v", path, err)
	}

	return pkgs, nil
}
