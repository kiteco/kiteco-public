package pythoncode

import (
	"log"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

// MockSignaturePatterns returns a mock SignaturePatterns object
// that can respond to requests for the provided method.
func MockSignaturePatterns(graph *pythonimports.Graph, methods ...string) *SignaturePatterns {
	opts := DefaultSignatureOptions
	opts.Coverage = 1.
	opts.MinUsage = 0.
	opts.MaxSignatures = 10
	sigPatterns := &SignaturePatterns{
		opts:           opts,
		graph:          graph,
		signatureIndex: make(map[int64]*MethodPatterns),
	}
	for _, method := range methods {
		node, err := graph.Find(method)
		if err != nil {
			log.Fatalln("could not find node for method:", method)
		}
		patterns := &MethodPatterns{
			Method: method,
			Kwargs: make(map[string]*ArgStats),
		}
		for i := 0; i < 4; i++ {
			patterns.Patterns = append(patterns.Patterns, &SignaturePattern{
				Args: i,
			})
			patterns.Args = append(patterns.Args, NewArgStats())
		}

		ProcessPatterns(patterns)

		sigPatterns.signatureIndex[node.ID] = patterns
	}
	return sigPatterns
}

// MockGithubPrior returns a empty github prior.
func MockGithubPrior() *GithubPrior {
	counts := make(map[string]int)
	counts["json.dumps"] = 10
	counts["json.dump"] = 20

	prior, _ := NewPackagePriorFromUniqueNameCounts("json", counts)

	stats := make(map[string]*PackagePrior)
	stats["json"] = prior

	return &GithubPrior{
		stats: stats,
	}
}
