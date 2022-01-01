package javascript

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertIndent(t *testing.T, tc string, expectedIndent int) {
	assert.Equal(t, expectedIndent, FindIndentation(tc))
}

func TestIndentInspect_Basic(t *testing.T) {
	src := `
const receiveProducts = products => ({
  type: types.RECEIVE_PRODUCTS,
  products
})
`

	assertIndent(t, src, 2)
}

func TestIndentInspect_Tab(t *testing.T) {
	src := `
const receiveProducts = products => ({
		type: types.RECEIVE_PRODUCTS,
		products
})
`

	assertIndent(t, src, -2)
}

func TestIndentInspect_ErrorNode(t *testing.T) {
	src := `
const receiveProducts = products => ({
  type: types.RECEIVE_PRODUCTS,
  products
`

	assertIndent(t, src, 2)
}

func TestIndentInspect_ZeroDepth(t *testing.T) {
	src := `
^const receiveProducts = products => ({})
`

	assertIndent(t, src, 0)
}
