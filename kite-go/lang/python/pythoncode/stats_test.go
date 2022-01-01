package pythoncode

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mockPrior() *PackagePrior {
	nameToNode := make(map[string]*Node)
	nameToNode["test"] = &Node{
		name: "test",
	}

	return &PackagePrior{
		nameToNode: nameToNode,
		idToNames:  make(map[int64][]string),
		root:       nameToNode["test"],
	}
}

func TestFindParent(t *testing.T) {
	prior := mockPrior()

	prior.findParent("test.numpy.array.array")
	act, _ := prior.findParent("test.numpy.array")
	assert.Equal(t, 3, len(prior.nameToNode))
	assert.Equal(t, prior.nameToNode["test.numpy"], act)
}

func TestInsert(t *testing.T) {
	prior := mockPrior()
	prior.insert("test.numpy.array.array", 40)

	assert.Equal(t, 4, len(prior.nameToNode))
	assert.Equal(t, 40, prior.nameToNode["test.numpy.array.array"].count)
	assert.Equal(t, 0, prior.nameToNode["test.numpy"].count)
}

func TestBackwardPropogate(t *testing.T) {
	prior := mockPrior()
	prior.insert("test.numpy.array.array", 40)
	prior.insert("test.numpy.array.transpose", 30)
	prior.insert("test.numpy", 17)

	assert.Equal(t, 5, len(prior.nameToNode))

	backwardPropogate(prior.root)

	assert.Equal(t, 17, prior.nameToNode["test"].count)
	assert.Equal(t, 17, prior.nameToNode["test.numpy"].count)
	assert.Equal(t, 70, prior.nameToNode["test.numpy.array"].count)
	assert.Equal(t, 40, prior.nameToNode["test.numpy.array.array"].count)
}

func TestForwardNormalize(t *testing.T) {
	prior := mockPrior()
	prior.insert("test.numpy.array.array", 40)
	prior.insert("test.numpy.array.transpose", 40)
	prior.insert("test.numpy", 17)
	prior.insert("test.test", 17)

	backwardPropogate(prior.root)
	forwardNormalize(prior.root)

	assert.EqualValues(t, 0, prior.nameToNode["test"].chainedlogProb)
	assert.Equal(t, math.Log(0.5), prior.nameToNode["test.numpy"].chainedlogProb)
	assert.Equal(t, math.Log(0.5), prior.nameToNode["test.numpy.array"].chainedlogProb)
	assert.Equal(t, math.Log(0.25), prior.nameToNode["test.numpy.array.transpose"].chainedlogProb)
}

func TestEntityLogProb(t *testing.T) {
	prior := mockPrior()
	prior.insert("test.numpy.array.array", 40)
	prior.insert("test.numpy.array.transpose", 40)
	prior.insert("test.numpy", 17)
	prior.insert("test.test", 17)

	backwardPropogate(prior.root)
	forwardNormalize(prior.root)

	prior.idToNames[0] = []string{"test"}
	prior.idToNames[1] = []string{"test.numpy"}
	prior.idToNames[2] = []string{"test.test"}
	prior.idToNames[3] = []string{"test.numpy.array"}
	prior.idToNames[4] = []string{"test.numpy.array.transpose", "test.numpy.array.array"}

	probs := prior.EntityChainedLogProbs()

	assert.Equal(t, math.Log(0.5), probs["test.numpy.array.transpose"])
}

func TestNewPriorFromUniqueNameCounts(t *testing.T) {
	identCounts := map[string]int{
		"test.numpy.array.array":     40,
		"test.numpy.array.transpose": 40,
		"test.numpy":                 17,
		"test.test":                  17,
	}

	prior, _ := NewPackagePriorFromUniqueNameCounts("test", identCounts)

	assert.Equal(t, []string{"test"}, prior.idToNames[0])
	probs := prior.EntityChainedLogProbs()

	assert.Equal(t, math.Log(0.25), probs["test.numpy.array.transpose"])
}
