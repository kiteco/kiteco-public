package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOntologyAdd(t *testing.T) {
	ont := newOntology()
	ont.add("min", "numpy.ma.core.min")
	ont.add("min", "numpy.ma.core.MaskedConstant.min")
	ont.add("min", "numpy.ma.core.min")

	assert.Equal(t, 2, len(ont.namespace["min"]))
}

func TestFindFuncCandidates(t *testing.T) {
	detector := detector{}

	content := `
	    numpy.testing.min()
	    test.123func
	    max()
	`

	candidates := detector.findFuncCandidates(content)

	assert.Equal(t, "numpy.testing.min", candidates[0])
	assert.Equal(t, "test.123func", candidates[1])
	assert.Equal(t, "max", candidates[2])
}
