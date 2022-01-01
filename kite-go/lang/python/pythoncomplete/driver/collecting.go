package driver

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

// CompletionTree is the tree representation of collection of completion
type CompletionTree struct {
	Completion Completion
	children   []*CompletionTree
}

type collectedNode struct {
	node        *CompletionTree
	targetState data.SelectedBuffer
}

// NewCompletionTree makes a new CompletionTree node
func NewCompletionTree(content Completion) *CompletionTree {
	return &CompletionTree{
		Completion: content,
	}
}

func (ct *CompletionTree) addChild(newComp Completion) *CompletionTree {
	newNode := NewCompletionTree(newComp)
	ct.addChildNode(newNode)
	return newNode
}

func (ct *CompletionTree) addChildNode(newNode *CompletionTree) {
	ct.children = append(ct.children, newNode)
}

func (m *Mixer) isRoot(node *CompletionTree) bool {
	return m.root == node
}

func findAndReplace(orderList []pythonproviders.Provider, target pythonproviders.Provider, replace pythonproviders.Provider) []pythonproviders.Provider {
	var key int
	for i, p := range orderList {
		if p == target {
			key = i
			break
		}
	}
	var res []pythonproviders.Provider
	res = append(res, orderList...)
	res[key] = replace
	return res
}

// collectCompletions does a BFS exploration of the buffer state of the scheduler
// For each state it compose the completion with its parent (to collect completions that can all be applied to the user buffer state)
// There's also a simple dedup of the completion based on the target buffer state and the list of placeholder
// (so 2 completions needs to have the same output state and the same set of placeholder to be considered as identical)
// Output order is random
func (m *Mixer) collectCompletions(sb data.SelectedBuffer) *CompletionTree {
	collectedStates := make(map[data.SelectedBufferHash]struct{})
	root := NewCompletionTree(Completion{})
	m.root = root
	candidates := []collectedNode{{
		node:        root,
		targetState: sb,
	}}
	var cand collectedNode
	atRoot := true
	for len(candidates) > 0 {
		cand, candidates = candidates[0], candidates[1:]
		if _, ok := collectedStates[cand.targetState.Hash()]; ok {
			continue
		}
		st := m.s.get(cand.targetState)
		collectedStates[cand.targetState.Hash()] = struct{}{}
		order := prioritizedSpeculationProviders
		// add GGNNModel Provider in the list of order
		if m.options.GGNNSubtokenEnabled {
			order = findAndReplace(order, pythonproviders.CallModel{}, pythonproviders.GGNNModel{})
		}
		if atRoot {
			order = prioritizedProviders
			if m.options.GGNNSubtokenEnabled {
				order = findAndReplace(order, pythonproviders.CallModel{}, pythonproviders.GGNNModel{})
			}

		}
		for _, k := range order {
			ps, ok := st.provisions[k]
			if !ok {
				continue
			}
			for _, cc := range ps.completions {
				for _, c := range cc {
					// TO FUTURE CONTRIBUTORS:
					// We want to do a minor refactor here: track the path from the completion to the root.
					// Instead of precomputing data for sorting, rendering, etc, having the full path
					// will enable this logic to move closer to the point of use.
					// see https://github.com/kiteco/kiteco/pull/9261#issuecomment-541165035
					if !atRoot {
						if c.Meta.MixingMeta.DoNotCompose {
							continue
						}
						c.Meta.Completion = c.Meta.Completion.MustAfter(cand.node.Completion.Meta.Completion)
						// Propagate the provider information from the parent completion
						c.Meta.Provider = cand.node.Completion.Meta.Provider
						if c.Meta.RenderMeta.Referent == nil {
							c.Meta.RenderMeta.Referent = cand.node.Completion.Meta.RenderMeta.Referent
						}
						if cand.node.Completion.Meta.Score > 0 {
							c.Meta.Score = c.Meta.Score * cand.node.Completion.Meta.Score
						}
					}
					newNode := cand.node.addChild(c)
					for _, tb := range c.speculate() {
						candidates = append(candidates, collectedNode{
							node:        newNode,
							targetState: tb,
						})
					}
				}
			}
		}
		atRoot = false
	}
	return root
}
