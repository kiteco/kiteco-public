package cooccurence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func assertModules(t *testing.T, src string, expected map[string]bool) {
	modules, err := ExtractModules([]byte(src))
	require.NoError(t, err)

	seen := make(map[string]bool)
	for _, module := range modules {
		switch {
		case !expected[module]:
			t.Errorf("extra module %s found\n", module)
		case seen[module]:
			t.Errorf("duplicate module %s found\n", module)
		default:
			seen[module] = true
		}
	}

	for module := range expected {
		if !seen[module] {
			t.Errorf("missing co-occurences for module %s\n", module)
		}
	}
}

func assertCooccurences(t *testing.T, src string, expected map[string]bool) {
	modules, err := ExtractModules([]byte(src))
	require.NoError(t, err)
	cooccurs := Cooccurences(modules)

	seen := make(map[string]bool)
	for _, cooccur := range cooccurs {
		if _, found := expected[cooccur.Module]; !found {
			t.Errorf("extra co-occurence for %s found\n", cooccur.Module)
			continue
		}
		seen[cooccur.Module] = true

		seenThisModule := make(map[string]bool)
		for _, module := range cooccur.Cooccuring {
			switch {
			case seenThisModule[module]:
				t.Errorf("duplicate co-occurence %s for module %s\n", module, cooccur.Module)
			case !expected[module]:
				t.Errorf("extra co-occurence %s for module %s\n", module, cooccur.Module)
			default:
				seenThisModule[module] = true
			}
		}
	}

	for module := range expected {
		if !seen[module] {
			t.Errorf("missing co-occurences for module %s\n", module)
		}
	}
}

func TestExtractModules(t *testing.T) {
	src := `
import foo, bar, star
from car import far, mar
import blue

class Foo():
	import hello.world
	from world import hello
`

	expected := map[string]bool{
		"foo":   true,
		"bar":   true,
		"star":  true,
		"car":   true,
		"blue":  true,
		"hello": true,
		"world": true,
	}

	assertModules(t, src, expected)
	assertCooccurences(t, src, expected)
}
