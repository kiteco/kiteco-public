package titleparser

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPreProcessTitle tests preprocessTitle.
func TestPreProcessTitle(t *testing.T) {
	test := "construct a 1D array [ of type `int16`]"
	target1 := "construct a 1D array of type int16"
	target2 := "construct a 1D array [ of type int16]"

	ret1, ret2 := preprocessTitle(test)
	assert.Equal(t, target1, ret1)
	assert.Equal(t, target2, ret2)
}

// TestImportedMethods tests importedMethods
func TestImportedMethods(t *testing.T) {
	prelude1 := "import os"
	prelude2 := "from os import path"
	prelude3 := "from os import path as p"
	prelude4 := "from os import path, sys as s"

	ret1 := importedMethods(prelude1)
	ret2 := importedMethods(prelude2)
	ret3 := importedMethods(prelude3)
	ret4 := importedMethods(prelude4)

	assert.Equal(t, 0, len(ret1))
	assert.Equal(t, "path", ret2[0])
	assert.Equal(t, "p", ret3[0])
	assert.Equal(t, "path", ret4[0])
	assert.Equal(t, "s", ret4[1])
}

// TestArgsOfFucs tests argsOfFuncs
func TestArgsOfFucs(t *testing.T) {
	// currently, argsOfFuncs does not return args of nested functions.
	exampleOf := []string{"array", "size"}

	code1 := `a = array(size(b), 3, dtype = 'float')`       // standard test case
	code2 := `a = array(size(b)`                            // unbalanced parentheses
	code3 := `return array(len(size(c)), c=5, dtype='int')` // nested scenario

	args1 := argsOfFuncs(code1, exampleOf)
	args2 := argsOfFuncs(code2, exampleOf)
	args3 := argsOfFuncs(code3, exampleOf)

	assert.Equal(t, 3, len(args1))
	assert.Equal(t, "size(b)", args1[0].val)
	assert.Equal(t, "dtype", args1[2].key)
	assert.Equal(t, "'float'", args1[2].val)

	assert.Equal(t, 0, len(args2))

	assert.Equal(t, 3, len(args3))
	assert.Equal(t, "len(size(c))", args3[0].val)
}

// TestMissingSpecs tests missingSpecs
func TestMissingSpecs(t *testing.T) {
	curated := `Construct a 1D array of type  int16   of size 5`
	standard := `Construct a 1D array [of type int16] [of size 5]`

	specs, _ := missingSpecs(curated, standard)
	assert.Equal(t, 2, len(specs))
	assert.Equal(t, "of type  int16", specs[0])
	assert.Equal(t, "of size 5", specs[1])

	// can't align
	curated = `Construct a 1D array of type 16`
	standard = `Construct a`
	specs, _ = missingSpecs(curated, standard)
	assert.Equal(t, 0, len(specs))
}

// TestSemanticCousinsSort tests the Sort function of type semanticCousins
func TestSemanticCousinsSort(t *testing.T) {
	var wordscores []*semanticCousin

	wordscores = append(wordscores, &semanticCousin{
		word:  "train",
		score: 0.5,
	})
	wordscores = append(wordscores, &semanticCousin{
		word:  "test",
		score: 1.0,
	})

	wordscores = append(wordscores, &semanticCousin{
		word:  "dev",
		score: 0.8,
	})

	sort.Sort(sort.Reverse(semanticCousins(wordscores)))

	assert.Equal(t, "test", wordscores[0].word)
}

// TestCosineSim tests cosineSim
func TestCosineSim(t *testing.T) {
	vec1 := []float64{1.0, 1.0, 1.0}
	vec2 := []float64{1.0, 1.0, 1.0}
	ret := cosineSim(vec1, vec2)
	exp := 1.0
	if ret-exp > 1e-8 {
		t.Errorf("cosineSim should be 1.0, got %f\n", ret)
	}

	vec2 = []float64{0, 0, 0}
	assert.Equal(t, -1.0, cosineSim(vec1, vec2))
}

// TestNumSentences tests numSentences
func TestNumSentences(t *testing.T) {
	test := `(ROOT (S (VP (VB Construct) (NP (NP (DT a) (NN matrix)) (PP (IN of) 
	(NP (NN int16)))) (PP (IN with) (NP (DT a) (JJ predefined) 
	(NN array)))) (. .))) (ROOT (NP (NP (NN Return)) (NP (DT the) (NN matrix)) (. .)))`

	assert.Equal(t, 2, numSentences(test))
}

// TestStartsWithBrackets tests startsWithBrackets
func TestHasBrackets(t *testing.T) {
	test1 := ` [3, 4, 1]`
	test2 := `3, 4, 5`
	test3 := `(3, 5)`
	test4 := `{3, 5}`

	assert.Equal(t, true, startsWithBrackets(test1))
	assert.Equal(t, false, startsWithBrackets(test2))
	assert.Equal(t, true, startsWithBrackets(test3))
	assert.Equal(t, true, startsWithBrackets(test4))
}

// TestFindPossession tests findPossession
func TestFindPossession(t *testing.T) {
	test := `(ROOT (S (VP (VB Specify) (NP (NP (DT the) (NN user) (POS 's)) (NN number)))))`
	tokens := parseTags(test)

	s, b := findPossession(tokens)
	assert.Equal(t, 5, len(tokens))
	assert.Equal(t, true, b)
	assert.Equal(t, "user's", s)
	assert.Equal(t, true, isVerb(tokens[0]))

}

// TestFindContraction tests findContraction
func TestFindContraction(t *testing.T) {
	test := `(ROOT (S (NP (NNP Should)) (VP (RB n't) (S (VP (VB specify) (NP (NP (DT the) (NN user) (POS 's)) (NN number)))))))`
	tokens := parseTags(test)

	s, b := findContraction(tokens)
	assert.Equal(t, 7, len(tokens))
	assert.Equal(t, true, b)
	assert.Equal(t, "Shouldn't", s)
}

// TestIsSpecialChar tests isSpecialChar
func TestIsSpecialChar(t *testing.T) {
	assert.Equal(t, false, isSpecialChar("[3, 4, 5]"))
	assert.Equal(t, true, isSpecialChar("["))
	assert.Equal(t, false, isSpecialChar(" "))
}

// TestVerbPhrases tests verbPhrases
func TestVerbPhrases(t *testing.T) {
	test := `(ROOT (S (VP (VB Specify) (NP (NP (DT the) (NN user) (POS 's)) (NN number)))))`
	vps := verbPhrases(test)

	assert.Equal(t, 1, len(vps))
	assert.Equal(t, "Specify the user's number", vps[0])

	test = `(ROOT (S (VP (VP (VB Construct) (NP (DT a) (NN matrix))) (CC and) (VP (VB return) (NP (DT the) (NN matrix))))))`
	vps = verbPhrases(test)
	assert.Equal(t, 3, len(vps))
}

// TestRemoveQuotes tests removeQuotes
func TestRemoveQuotes(t *testing.T) {
	test := "'float16'"
	assert.Equal(t, "float16", removeQuotes(test))

	test = "\"test\""
	assert.Equal(t, "test", removeQuotes(test))
}

// TestRemoveExtraSpace tests removeExtraSpace
func TestRemoveExtraSpace(t *testing.T) {
	test := `Construct an array of     int16  [with a matrix of size (5,  8) ]     `
	exp := `Construct an array of int16 [with a matrix of size (5, 8) ]`
	assert.Equal(t, exp, removeExtraSpace(test))
}

// TestStandardTitle tests expectedTitle
func TestStandardTitle(t *testing.T) {
	test := `(ROOT (S (VP (VB Construct) (NP (NP (DT a) (NN matrix)) 
	(PP (IN of) (NP (NN int16)))) (PP (IN with) (NP (DT a) (JJ predefined) (NN array))))))`

	exp := `Construct a matrix [of int16] [with a predefined array]`
	assert.Equal(t, exp, expectedTitle(test))

	test = "(ROOT (S (VP (VB Construct) (NP (NP (DT a) (NN matrix)) (PP (IN of) (NP (`` `) (NP (NN int16)) ('' ') (PP (IN with) (NP (DT a) (JJ predefined) (NN array)))))))))"
	exp = "Construct a matrix [of `int16'] [with a predefined array]"
	assert.Equal(t, exp, expectedTitle(test))
}

// TestSubstringOverlap tests substringOverlap
func TestSubstringOverlap(t *testing.T) {
	parse := "(ROOT (S (VP (VB Get) (SBAR (S (NP (NP (NN information)) (PP (IN about) (NP (DT the) (NNS arguments)))) (VP (VBD passed) (PP (IN into) (NP (DT a) (NN frame)))))))))"
	tags := parseTags(parse)
	target := "currentframe()"

	assert.Equal(t, true, substringOverlap(target, tags))

	parse = "(ROOT (S (VP (VBG Constructing) (NP (NP (CD one) (NN matrix)) (PP (IN of) (NP (NN int16)))) (PP (IN with) (NP (DT a) (JJ predefined) (NN array)))) (. .)))"
	tags = parseTags(parse)
	target = "dtype"
	assert.Equal(t, false, substringOverlap(target, tags))

	target = "float"
	assert.Equal(t, false, substringOverlap(target, tags))
}
