package driver

import (
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

// CompletionTree is the tree representation of collection of completion
type CompletionTree struct {
	completion completion
	children   []*CompletionTree
}

// NewCompletionTree makes a new CompletionTree node
func NewCompletionTree(content completion) *CompletionTree {
	return &CompletionTree{
		completion: content,
	}
}

func (ct *CompletionTree) addChild(newComp completion) *CompletionTree {
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

// collectCompletions does a BFS exploration of the buffer state of the scheduler
// For each state it compose the completion with its parent (to collect completions that can all be applied to the user buffer state)
// There's also a simple dedup of the completion based on the target buffer state and the list of placeholder
// (so 2 completions needs to have the same output state and the same set of placeholder to be considered as identical)
// Output order is random
func (m *Mixer) collectCompletions() *CompletionTree {
	collectedStates := make(map[data.SelectedBufferHash]struct{})
	root := NewCompletionTree(completion{})
	m.root = root
	candidates := []collectedNode{{
		node:        root,
		targetState: m.selectedBuffer,
	}}
	var cand collectedNode
	dedupSet := make(mixingSet)
	atRoot := true
	for len(candidates) > 0 {
		cand, candidates = candidates[0], candidates[1:]
		if _, ok := collectedStates[cand.targetState.Hash()]; ok {
			continue
		}
		st := m.s.get(cand.targetState)
		collectedStates[cand.targetState.Hash()] = struct{}{}
		order := prioritizedSpeculationProviders
		if atRoot {
			order = prioritizedProviders
		}
		for _, k := range order {
			ps, ok := st.provisions[k]
			if !ok {
				continue
			}
			for _, cc := range ps.completions {
				for _, c := range cc {
					if !atRoot {
						c.meta.Completion = c.meta.Completion.MustAfter(cand.node.completion.meta.Completion)
						// Propagate the provider information from the parent completion
						c.meta.Provider = cand.node.completion.meta.Provider
						if cand.node.completion.meta.Score > 0 {
							c.meta.Score = c.meta.Score * cand.node.completion.meta.Score
						}
					}
					if !dedupSet.add(c) {
						continue
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

type collectedNode struct {
	node        *CompletionTree
	targetState data.SelectedBuffer
}
