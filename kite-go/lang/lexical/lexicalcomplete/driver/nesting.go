package driver

import (
	"strings"
)

func (m *Mixer) nestCompletions(completions *CompletionTree) {
	nestingHosts := make(map[string]*CompletionTree)
	for _, c := range completions.children {
		t := c.completion.meta.Snippet.Text
		if len(t) == 0 {
			continue
		}
		if strings.IndexAny(t, "([{") == len(t)-1 {
			nestingHosts[t] = c
		}
	}
	newChildren := make([]*CompletionTree, 0, len(completions.children))
childrenLoop:
	for _, c := range completions.children {
		for prefix, host := range nestingHosts {
			if c == host {
				break
			}
			if strings.HasPrefix(c.completion.meta.Snippet.Text, prefix) {
				host.children = append(host.children, c)
				continue childrenLoop
			}
		}
		newChildren = append(newChildren, c)
	}
	completions.children = newChildren
}
