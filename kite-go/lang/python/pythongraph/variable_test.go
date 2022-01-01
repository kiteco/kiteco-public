package pythongraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertHasVariableID(t *testing.T, src string, addMissing bool, names ...string) {

	b := requireBuilderOpts(t, emptyRM(t), src, addMissing)

	for _, name := range names {
		ns := requireNamesForTest(t, name, b)
		for ne := range ns.Set() {
			assert.NotEqual(t, -1, b.vm.VariableIDFor(ne))
		}
	}
}

func assertNoVariableID(t *testing.T, src string, names ...string) {
	b := requireBuilderOpts(t, emptyRM(t), src, false)

	for _, name := range names {
		for _, v := range b.vm.Variables {
			for _, ne := range v.Refs.Names() {
				assert.NotEqual(t, name, ne)
			}
		}
	}
}

func TestNameToVariableID(t *testing.T) {
	src := `[x for x in []]`

	assertHasVariableID(t, src, false, "x")

}

func TestUndefinedVariable(t *testing.T) {
	src := `foo(x,y)`

	assertHasVariableID(t, src, true, "foo", "x", "y")

}

func TestNoVariable(t *testing.T) {
	src := `
x = 1

y
`

	assertHasVariableID(t, src, false, "x")
	assertNoVariableID(t, src, "y")

}
