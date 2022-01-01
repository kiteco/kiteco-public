package recommend

import (
	"math"
	"regexp"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type splitBlocksTC struct {
	content  string
	expected []Block
}

func TestSplitBlocks(t *testing.T) {
	tcs := []splitBlocksTC{
		splitBlocksTC{
			content: `alpha beta
gamma

delta epsilon zeta`,
			expected: []Block{
				Block{
					Content:   "alpha beta\ngamma",
					FirstLine: 1,
					LastLine:  2,
				},
				Block{
					Content:   "delta epsilon zeta",
					FirstLine: 4,
					LastLine:  4,
				},
			},
		},
		splitBlocksTC{
			content: `alpha beta
gamma

delta
epsilon
zeta

eta theta
iota
`,
			expected: []Block{
				Block{
					Content:   "alpha beta\ngamma",
					FirstLine: 1,
					LastLine:  2,
				},
				Block{
					Content:   "delta\nepsilon\nzeta",
					FirstLine: 4,
					LastLine:  6,
				},
				Block{
					Content:   "eta theta\niota",
					FirstLine: 8,
					LastLine:  9,
				},
			},
		},
		splitBlocksTC{
			content: `alpha beta
gamma
z
delta epsilon zeta`,
			expected: []Block{
				Block{
					Content:   "alpha beta\ngamma\nz",
					FirstLine: 1,
					LastLine:  3,
				},
				Block{
					Content:   "delta epsilon zeta",
					FirstLine: 4,
					LastLine:  4,
				},
			},
		},
		splitBlocksTC{
			content: `alpha beta
gamma
x
delta
epsilon
zeta

eta theta
iota
z
`,
			expected: []Block{
				Block{
					Content:   "alpha beta\ngamma\nx",
					FirstLine: 1,
					LastLine:  3,
				},
				Block{
					Content:   "delta\nepsilon\nzeta",
					FirstLine: 4,
					LastLine:  6,
				},
				Block{
					Content:   "eta theta\niota\nz",
					FirstLine: 8,
					LastLine:  10,
				},
			},
		},
	}
	for _, tc := range tcs {
		require.Equal(t, tc.expected, splitBlocks(tc.content))
	}
}

type findKeywordsTC struct {
	idf           map[shingle]float32
	opts          vectorizerOptions
	request       Request
	currentBuffer string
	inspectBlock  string
	expected      []string
}

func TestFindKeywords(t *testing.T) {
	tcs := []findKeywordsTC{
		findKeywordsTC{
			idf: map[shingle]float32{
				newShingle([]rune("alpha")): 10,
				newShingle([]rune("gamma")): 20,
				newShingle([]rune("delta")): 15,
				newShingle([]rune("epsil")): 1,
				newShingle([]rune("psilo")): 2,
				newShingle([]rune("silon")): 3,
			},
			opts: vectorizerOptions{
				shingleSize: 5,
			},
			request: Request{
				MaxBlockKeywords: -1,
			},
			currentBuffer: `
				alpha.beta()
				gamma.delta()
				epsilon()
				beta = gamma()
			`,
			inspectBlock: `
				alpha.beta()
				alpha.gamma(epsilon())
				phi.sigma(alpha.alpha())
				alpha = alpha(alpha.alpha())
			`,
			expected: []string{
				"gamma",
				"alpha",
				"epsilon",
			},
		},
		findKeywordsTC{
			idf: map[shingle]float32{
				newShingle([]rune("alpha")): 10,
				newShingle([]rune("gamma")): 20,
				newShingle([]rune("delta")): 15,
				newShingle([]rune("epsil")): 1,
				newShingle([]rune("psilo")): 2,
				newShingle([]rune("silon")): 3,
			},
			opts: vectorizerOptions{
				shingleSize: 5,
			},
			request: Request{
				MaxBlockKeywords: 2,
			},
			currentBuffer: `
				alpha.beta()
				gamma.delta()
				epsilon()
				beta = gamma()
			`,
			inspectBlock: `
				alpha.beta()
				alpha.gamma(epsilon())
				phi.sigma(alpha.alpha())
				alpha = alpha(alpha.alpha())
			`,
			expected: []string{
				"gamma",
				"alpha",
			},
		},
		findKeywordsTC{
			idf: map[shingle]float32{
				newShingle([]rune("alpha")): 10,
				newShingle([]rune("gamma")): 20,
				newShingle([]rune("delta")): 15,
				newShingle([]rune("epsil")): 1,
				newShingle([]rune("psilo")): 2,
				newShingle([]rune("silon")): 3,
			},
			opts: vectorizerOptions{
				shingleSize: 5,
			},
			request: Request{
				MaxBlockKeywords: 3,
			},
			currentBuffer: `
				alPHa.beta()
				GAMma.delta()
				epsilon()
				beta = GAMma()
			`,
			inspectBlock: `
				alPHa.beta()
				alPHa.GAMma(epsilon())
				phi.sigma(alPHa.alPHa())
				alpha_Gamma
				alPHa = alPHa(alPHa.alPHa())
			`,
			expected: []string{
				"alpha_Gamma",
				"GAMma",
				"alPHa",
			},
		},
	}

	for _, tc := range tcs {
		v := vectorizer{
			idf:         tc.idf,
			opts:        tc.opts,
			wordsRegexp: regexp.MustCompile(wordsRegexp),
		}
		cov := v.makeVector(tc.currentBuffer).toCovector()
		keywords := v.findKeywords(tc.inspectBlock, cov, tc.request)
		var words []string
		for _, keyword := range keywords {
			words = append(words, keyword.Word)
		}
		require.Equal(t, tc.expected, words)
		for i, keyword := range keywords {
			if i == 0 {
				continue
			}
			require.True(t, keyword.Score <= keywords[i-1].Score)
		}
	}
}

type newShingleTC struct {
	rs       []rune
	expected shingle
}

func TestNewShingle(t *testing.T) {
	tcs := []newShingleTC{
		newShingleTC{
			rs:       nil,
			expected: 0,
		},
		newShingleTC{
			rs:       []rune{'a'},
			expected: 0,
		},
		newShingleTC{
			rs:       []rune{'b', 'a'},
			expected: 0x20,
		},
		newShingleTC{
			rs:       []rune{'c', 'b', 'a'},
			expected: 0x820,
		},
		newShingleTC{
			rs:       []rune{'d', 'c', 'b', 'a'},
			expected: 0x18820,
		},
		newShingleTC{
			rs:       []rune{'e', 'd', 'c', 'b', 'a'},
			expected: 0x418820,
		},
		newShingleTC{
			rs:       []rune{'f', 'e', 'd', 'c', 'b', 'a'},
			expected: 0xa418820,
		},
		newShingleTC{
			rs:       []rune{'g', 'f', 'e', 'd', 'c', 'b', 'a'},
			expected: 0x8a418820,
		},
		newShingleTC{
			rs:       []rune{'h', 'g', 'f', 'e', 'd', 'c', 'b', 'a'},
			expected: 0x8a418820,
		},
		newShingleTC{
			rs:       []rune{'e', 'd', 'c', 'b', 0},
			expected: 0x41883a,
		},
		newShingleTC{
			rs:       []rune{'e', 'd', 'c', 'b', 5},
			expected: 0x41883f,
		},
		newShingleTC{
			rs:       []rune{'e', 'd', 'c', 'b', 6},
			expected: 0x41883a,
		},
	}

	for _, tc := range tcs {
		require.Equal(t, tc.expected, newShingle(tc.rs))
	}
}

type countShingleTC struct {
	content         string
	shingleSize     int
	keepUnderscores bool
	expected        map[shingle]int
}

func TestCountShingle(t *testing.T) {
	tcs := []countShingleTC{
		countShingleTC{
			content: `
alpha beta
GAMMA(Delta_Epsilon, phi)
	zeta zALPHA
`,
			shingleSize: 5,
			expected: map[shingle]int{
				newShingle([]rune("alpha")): 2,
				newShingle([]rune("gamma")): 1,
				newShingle([]rune("delta")): 1,
				newShingle([]rune("epsil")): 1,
				newShingle([]rune("psilo")): 1,
				newShingle([]rune("silon")): 1,
				newShingle([]rune("zalph")): 1,
				newShingle([]rune("eltae")): 1,
				newShingle([]rune("ltaep")): 1,
				newShingle([]rune("taeps")): 1,
				newShingle([]rune("aepsi")): 1,
			},
		},
		countShingleTC{
			content: `
alpha beta
GAMMA(Delta_Epsilon, phi)
	zeta zALPHA
`,
			shingleSize:     5,
			keepUnderscores: true,
			expected: map[shingle]int{
				newShingle([]rune("alpha")): 2,
				newShingle([]rune("gamma")): 1,
				newShingle([]rune("delta")): 1,
				newShingle([]rune("epsil")): 1,
				newShingle([]rune("psilo")): 1,
				newShingle([]rune("silon")): 1,
				newShingle([]rune("zalph")): 1,
			},
		},
		countShingleTC{
			content:     "çöêøåñéù êøåñé",
			shingleSize: 4,
			expected: map[shingle]int{
				newShingle([]rune("çöêø")): 1,
				newShingle([]rune("öêøå")): 1,
				newShingle([]rune("êøåñ")): 2,
				newShingle([]rune("øåñé")): 2,
				newShingle([]rune("åñéù")): 1,
			},
		},
		countShingleTC{
			content:         "çöêøåñéù êøåñé",
			shingleSize:     4,
			keepUnderscores: true,
			expected: map[shingle]int{
				newShingle([]rune("çöêø")): 1,
				newShingle([]rune("öêøå")): 1,
				newShingle([]rune("êøåñ")): 2,
				newShingle([]rune("øåñé")): 2,
				newShingle([]rune("åñéù")): 1,
			},
		},
	}
	for _, tc := range tcs {
		require.Equal(t, tc.expected, countShingles(tc.content, tc.shingleSize, tc.keepUnderscores))
	}
}

type addTC struct {
	counter  counter
	content  string
	expected counter
}

func TestAdd(t *testing.T) {
	tcs := []addTC{
		addTC{
			counter: counter{
				shingleSize: 4,
				size:        17,
				counts: map[shingle]int{
					newShingle([]rune("alph")): 11,
					newShingle([]rune("lpha")): 8,
					newShingle([]rune("beta")): 23,
					newShingle([]rune("zeta")): 19,
				},
			},
			content: "alpha beta gamma",
			expected: counter{
				shingleSize: 4,
				size:        18,
				counts: map[shingle]int{
					newShingle([]rune("alph")): 12,
					newShingle([]rune("lpha")): 9,
					newShingle([]rune("beta")): 24,
					newShingle([]rune("gamm")): 1,
					newShingle([]rune("amma")): 1,
					newShingle([]rune("zeta")): 19,
				},
			},
		},
		addTC{
			counter: counter{
				shingleSize: 4,
				size:        17,
				counts: map[shingle]int{
					newShingle([]rune("alph")): 11,
					newShingle([]rune("lpha")): 8,
					newShingle([]rune("beta")): 23,
					newShingle([]rune("zeta")): 19,
				},
			},
			content: "alpha beta gamma alpha beta alph lpha",
			expected: counter{
				shingleSize: 4,
				size:        18,
				counts: map[shingle]int{
					newShingle([]rune("alph")): 12,
					newShingle([]rune("lpha")): 9,
					newShingle([]rune("beta")): 24,
					newShingle([]rune("gamm")): 1,
					newShingle([]rune("amma")): 1,
					newShingle([]rune("zeta")): 19,
				},
			},
		},
		addTC{
			counter: counter{
				shingleSize: 4,
				size:        17,
				counts: map[shingle]int{
					newShingle([]rune("alph")): 11,
					newShingle([]rune("lpha")): 8,
					newShingle([]rune("beta")): 23,
					newShingle([]rune("zeta")): 19,
				},
			},
			content: "",
			expected: counter{
				shingleSize: 4,
				size:        18,
				counts: map[shingle]int{
					newShingle([]rune("alph")): 11,
					newShingle([]rune("lpha")): 8,
					newShingle([]rune("beta")): 23,
					newShingle([]rune("zeta")): 19,
				},
			},
		},
	}
	for _, tc := range tcs {
		tc.counter.add(tc.content)
		require.Equal(t, tc.expected, tc.counter)
	}
}

type newVectorizerTC struct {
	counter     counter
	expectedIdf map[shingle]float32
}

func TestNewVectorizer(t *testing.T) {
	tcs := []newVectorizerTC{
		newVectorizerTC{},
		newVectorizerTC{
			counter: counter{
				counts: map[shingle]int{
					newShingle([]rune("alpha")): 3,
					newShingle([]rune("beta")):  8,
					newShingle([]rune("gamma")): 11,
					newShingle([]rune("delta")): 1,
				},
				size: 15,
			},
			expectedIdf: map[shingle]float32{
				newShingle([]rune("alpha")): 1.386,
				newShingle([]rune("delta")): 2.639,
			},
		},
	}
	for _, tc := range tcs {
		vectorizer := tc.counter.newVectorizer()
		require.InDeltaMapValues(t, tc.expectedIdf, vectorizer.idf, 1e-3)
		require.NotNil(t, vectorizer.vectorSet)
		require.NotNil(t, vectorizer.wordsRegexp)
	}
}

type shingleDotTC struct {
	cov      map[shingle]float32
	vec      []valuedShingle
	expected float32
}

func TestShingleDot(t *testing.T) {
	tcs := []shingleDotTC{
		shingleDotTC{
			cov: map[shingle]float32{
				newShingle([]rune("alpha")): 1.5,
				newShingle([]rune("beta")):  0.5,
				newShingle([]rune("gamma")): 3.5,
			},
			vec: []valuedShingle{
				newValuedShingle(newShingle([]rune("beta")), 4.5),
				newValuedShingle(newShingle([]rune("gamma")), 5.5),
				newValuedShingle(newShingle([]rune("delta")), 3.5),
			},
			expected: 21.5,
		},
		shingleDotTC{
			vec: []valuedShingle{
				newValuedShingle(newShingle([]rune("beta")), 4.5),
				newValuedShingle(newShingle([]rune("gamma")), 5.5),
				newValuedShingle(newShingle([]rune("delta")), 3.5),
			},
			expected: 0,
		},
		shingleDotTC{
			cov: map[shingle]float32{
				newShingle([]rune("alpha")): 1.5,
				newShingle([]rune("beta")):  0.5,
				newShingle([]rune("gamma")): 3.5,
			},
			expected: 0,
		},
		shingleDotTC{
			expected: 0,
		},
	}
	for _, tc := range tcs {
		require.Equal(t, tc.expected, shingleDot(tc.cov, tc.vec))
	}
}

type shingleVectorToCovectorTC struct {
	vec      shingleVector
	expected shingleCovector
}

func TestShingleVectorToCovector(t *testing.T) {
	tcs := []shingleVectorToCovectorTC{
		shingleVectorToCovectorTC{
			vec: shingleVector{
				coords: []valuedShingle{
					newValuedShingle(newShingle([]rune("alpha")), 3),
					newValuedShingle(newShingle([]rune("beta")), 4),
				},
				norm: 5,
			},
			expected: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("alpha")): 3,
					newShingle([]rune("beta")):  4,
				},
				norm: 5,
			},
		},
	}

	for _, tc := range tcs {
		require.Equal(t, tc.expected, tc.vec.toCovector())
	}
}

type toVectorAndToCovectorTC struct {
	vec []valuedShingle
	cov map[shingle]float32
}

func TestToVectorAndToCovector(t *testing.T) {
	tcs := []toVectorAndToCovectorTC{
		toVectorAndToCovectorTC{
			vec: []valuedShingle{
				newValuedShingle(newShingle([]rune("alpha")), 3),
				newValuedShingle(newShingle([]rune("beta")), 4),
			},
			cov: map[shingle]float32{
				newShingle([]rune("alpha")): 3,
				newShingle([]rune("beta")):  4,
			},
		},
	}

	for _, tc := range tcs {
		require.Equal(t, tc.cov, toCovector(tc.vec))
		require.Equal(t, tc.cov, toCovector(toVector(tc.cov)))
		require.ElementsMatch(t, tc.vec, toVector(tc.cov))
		require.ElementsMatch(t, tc.vec, toVector(toCovector(tc.vec)))
	}
}

type vectorNormTC struct {
	vec      []valuedShingle
	expected float32
}

func TestVectorNorm(t *testing.T) {
	tcs := []vectorNormTC{
		vectorNormTC{
			vec: []valuedShingle{
				newValuedShingle(newShingle([]rune("alpha")), 3),
				newValuedShingle(newShingle([]rune("beta")), 4),
			},
			expected: 5,
		},
		vectorNormTC{
			vec: []valuedShingle{
				newValuedShingle(newShingle([]rune("alpha")), 3),
			},
			expected: 3,
		},
		vectorNormTC{},
	}
	for _, tc := range tcs {
		require.Equal(t, tc.expected, vectorNorm(tc.vec))
	}
}

type covectorNormTC struct {
	cov      map[shingle]float32
	expected float32
}

func TestCovectorNorm(t *testing.T) {
	tcs := []covectorNormTC{
		covectorNormTC{
			cov: map[shingle]float32{
				newShingle([]rune("alpha")): 3,
				newShingle([]rune("beta")):  4,
			},
			expected: 5,
		},
		covectorNormTC{
			cov: map[shingle]float32{
				newShingle([]rune("alpha")): 3,
			},
			expected: 3,
		},
		covectorNormTC{},
	}
	for _, tc := range tcs {
		require.Equal(t, tc.expected, covectorNorm(tc.cov))
	}
}

type recommendBlocksTC struct {
	vectorizer vectorizer
	base       string
	inspect    string
	request    Request
	expected   []Block
}

func TestRecommendBlocks(t *testing.T) {
	tcs := []recommendBlocksTC{
		recommendBlocksTC{
			vectorizer: vectorizer{
				idf: map[shingle]float32{
					newShingle([]rune("alpha")): 4,
					newShingle([]rune("gamma")): 2,
					newShingle([]rune("delta")): 0,
					newShingle([]rune("epsil")): 15,
					newShingle([]rune("psilo")): 15,
					newShingle([]rune("silon")): 15,
				},
				opts: vectorizerOptions{
					shingleSize: 5,
				},
				wordsRegexp: regexp.MustCompile(wordsRegexp),
			},
			base: `
alpha(beta)
alpha.beta
beta(delta)

delta.epsilon
delta(gamma.delta)
`,
			inspect: `
gamma.beta

epsilon(alpha)


beta.gamma
delta(epsilon)

gamma.beta
`,
			request: Request{
				MaxBlockKeywords: 3,
			},
			expected: []Block{
				Block{
					Content:   "epsilon(alpha)",
					FirstLine: 4,
					LastLine:  4,
					Keywords: []Keyword{
						Keyword{Word: "epsilon"},
						Keyword{Word: "alpha"},
					},
				},
				Block{
					Content:   "beta.gamma\ndelta(epsilon)",
					FirstLine: 7,
					LastLine:  8,
					Keywords: []Keyword{
						Keyword{Word: "epsilon"},
						Keyword{Word: "gamma"},
					},
				},
				Block{
					Content:   "gamma.beta",
					FirstLine: 2,
					LastLine:  2,
					Keywords: []Keyword{
						Keyword{Word: "gamma"},
					},
				},
				Block{
					Content:   "gamma.beta",
					FirstLine: 10,
					LastLine:  10,
					Keywords: []Keyword{
						Keyword{Word: "gamma"},
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		actual, err := tc.vectorizer.recommendBlocks(tc.base, tc.inspect, tc.request)
		require.NoError(t, err)
		for i := range actual {
			// Don't test the block Probability values
			actual[i].Probability = 0
			for j := range actual[i].Keywords {
				// Don't test the keyword Score values
				actual[i].Keywords[j].Score = 0
			}
		}
		require.Equal(t, tc.expected, actual)
	}
}

type curateLocalContentTC struct {
	content       string
	currentLine   int
	local         localization
	expected      string
	expectedError error
}

func TestCurateLocalContent(t *testing.T) {
	tcs := []curateLocalContentTC{
		curateLocalContentTC{
			content: `
alpha
beta
gamma
delta
epsilon

alpha
beta

gamma
delta
epsilon
`,
			currentLine: 8,
			local:       localization{size: 3},
			expected: `epsilon


alpha
alpha
alpha
beta
beta
`,
		},
		curateLocalContentTC{
			content: `
alpha
beta
gamma
delta
epsilon

alpha
beta

gamma
delta
epsilon
`,
			currentLine: 2,
			local:       localization{size: 4},
			expected: `


alpha
alpha
alpha
alpha
beta
beta
beta
gamma
gamma
delta`,
		},
		curateLocalContentTC{
			content: `
alpha
beta
gamma
delta
epsilon

alpha
beta

gamma
delta
epsilon
`,
			currentLine:   20,
			local:         localization{size: 10},
			expectedError: ErrInvalidCurrentLine,
		},
	}
	for _, tc := range tcs {
		actual, err := curateLocalContent(tc.content, tc.currentLine, tc.local)
		require.Equal(t, tc.expectedError, err)
		require.Equal(t, tc.expected, actual)
	}
}

type uniformVectorizeTC struct {
	vectorizer vectorizer
	content    string
	expected   shingleVector
}

func TestUniformVectorize(t *testing.T) {
	tcs := []uniformVectorizeTC{
		uniformVectorizeTC{
			vectorizer: vectorizer{
				idf: map[shingle]float32{
					newShingle([]rune("alpha")): 3,
					newShingle([]rune("gamma")): 5,
				},
				opts: vectorizerOptions{
					shingleSize: 5,
				},
			},
			content: "alpha(gamma.beta) + gamma(delta)",
			expected: shingleVector{
				coords: []valuedShingle{
					newValuedShingle(newShingle([]rune("alpha")), float32(math.Log(2)*3)),
					newValuedShingle(newShingle([]rune("gamma")), float32(math.Log(3)*5)),
				},
			},
		},
	}
	for _, tc := range tcs {
		tc.expected.norm = vectorNorm(tc.expected.coords)
		actual := tc.vectorizer.makeVector(tc.content)
		require.Equal(t, tc.expected.norm, actual.norm)
		require.Equal(t, tc.expected.modTime, actual.modTime)
		require.ElementsMatch(t, tc.expected.coords, actual.coords)
	}
}

type mixCovectorsTC struct {
	globalCovector shingleCovector
	localCovector  shingleCovector
	local          localization
	expected       shingleCovector
}

func TestMixCovectors(t *testing.T) {
	tcs := []mixCovectorsTC{
		mixCovectorsTC{
			globalCovector: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("alpha")): 3,
					newShingle([]rune("beta")):  4,
				},
				norm: 5,
			},
			localCovector: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("beta")):  400,
					newShingle([]rune("gamma")): 300,
				},
				norm: 500,
			},
			local: localization{weight: 0.6},
			expected: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("alpha")): 0.24,
					newShingle([]rune("beta")):  0.8,
					newShingle([]rune("gamma")): 0.36,
				},
			},
		},
		mixCovectorsTC{
			globalCovector: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("alpha")): 3,
					newShingle([]rune("beta")):  4,
				},
				norm: 5,
			},
			local: localization{weight: 0.6},
			expected: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("alpha")): 0.24,
					newShingle([]rune("beta")):  0.32,
				},
			},
		},
		mixCovectorsTC{
			localCovector: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("beta")):  400,
					newShingle([]rune("gamma")): 300,
				},
				norm: 500,
			},
			local: localization{weight: 0.6},
			expected: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("beta")):  0.48,
					newShingle([]rune("gamma")): 0.36,
				},
			},
		},
		mixCovectorsTC{
			globalCovector: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("alpha")): 3,
					newShingle([]rune("beta")):  4,
				},
				norm: 5,
			},
			localCovector: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("beta")):  400,
					newShingle([]rune("gamma")): 300,
				},
				norm: 500,
			},
			local: localization{weight: 1},
			expected: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("alpha")): 0,
					newShingle([]rune("beta")):  0.8,
					newShingle([]rune("gamma")): 0.6,
				},
			},
		},
		mixCovectorsTC{
			globalCovector: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("alpha")): 3,
					newShingle([]rune("beta")):  4,
				},
				norm: 5,
			},
			localCovector: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("beta")):  400,
					newShingle([]rune("gamma")): 300,
				},
				norm: 500,
			},
			local: localization{weight: 0},
			expected: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("alpha")): 0.6,
					newShingle([]rune("beta")):  0.8,
					newShingle([]rune("gamma")): 0,
				},
			},
		},
	}
	for _, tc := range tcs {
		tc.expected.norm = covectorNorm(tc.expected.coords)
		actual := mixCovectors(tc.globalCovector, tc.localCovector, tc.local)
		require.InDeltaMapValues(t, tc.expected.coords, actual.coords, 1e-6)
		require.Equal(t, tc.expected.norm, actual.norm)
	}
}

type scoreTC struct {
	vectorizer vectorizer
	cov        shingleCovector
	vec        shingleVector
	expected   float32
}

func TestScore(t *testing.T) {
	tcs := []scoreTC{
		scoreTC{
			vectorizer: vectorizer{
				opts: vectorizerOptions{scoreRegularization: 1},
			},
			vec: shingleVector{
				coords: []valuedShingle{
					newValuedShingle(newShingle([]rune("alpha")), 1),
					newValuedShingle(newShingle([]rune("beta")), 10),
				},
				norm: 9,
			},
			cov: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("beta")):  20,
					newShingle([]rune("gamma")): 30,
				},
				norm: 10,
			},
			expected: 2,
		},
		scoreTC{
			vectorizer: vectorizer{
				opts: vectorizerOptions{scoreRegularization: 1},
			},
			vec: shingleVector{
				coords: []valuedShingle{
					newValuedShingle(newShingle([]rune("alpha")), 10),
					newValuedShingle(newShingle([]rune("beta")), 1),
				},
				norm: 9,
			},
			cov: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("beta")):  20,
					newShingle([]rune("gamma")): 30,
				},
				norm: 10,
			},
			expected: 0.2,
		},
		scoreTC{
			vectorizer: vectorizer{
				opts: vectorizerOptions{scoreRegularization: 1},
			},
			vec: shingleVector{
				coords: []valuedShingle{
					newValuedShingle(newShingle([]rune("alpha")), 10),
					newValuedShingle(newShingle([]rune("beta")), 1),
				},
				norm: 9,
			},
			cov: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("beta")):  20,
					newShingle([]rune("gamma")): 30,
				},
				norm: 0,
			},
			expected: 0,
		},
	}
	for _, tc := range tcs {
		actual := tc.vectorizer.score(tc.cov, tc.vec)
		require.InDelta(t, tc.expected, actual, 1e-6)
	}
}

type recommendFilesFromVectorTC struct {
	vectorizer vectorizer
	currentID  fileID
	cov        shingleCovector
	request    Request
	expected   []File
}

func TestRecommendFilesFromVector(t *testing.T) {
	tcs := []recommendFilesFromVectorTC{
		recommendFilesFromVectorTC{
			vectorizer: vectorizer{
				vectorSet: vectorSet{
					data: map[fileID]shingleVector{
						10: shingleVector{
							coords: []valuedShingle{
								newValuedShingle(newShingle([]rune("epsil")), 3),
								newValuedShingle(newShingle([]rune("phi")), 4),
							},
							norm: 5,
						},
						20: shingleVector{
							coords: []valuedShingle{
								newValuedShingle(newShingle([]rune("epsil")), 3),
								newValuedShingle(newShingle([]rune("phi")), 4),
							},
							norm: 50,
						},
						30: shingleVector{
							coords: []valuedShingle{
								newValuedShingle(newShingle([]rune("epsil")), 3),
								newValuedShingle(newShingle([]rune("zeta")), 4),
							},
							norm: 5,
						},
					},
					m: new(sync.RWMutex),
				},
			},
			currentID: 3,
			cov: shingleCovector{
				coords: map[shingle]float32{
					newShingle([]rune("epsil")): 10,
					newShingle([]rune("zeta")):  200,
					newShingle([]rune("sigma")): 50,
				},
				norm: 10,
			},
			expected: []File{
				File{id: 30},
				File{id: 10},
				File{id: 20},
			},
		},
	}
	for _, tc := range tcs {
		actual := tc.vectorizer.recommendFilesFromCovector(tc.currentID, tc.cov, tc.request)
		for i := range actual {
			// Don't test the file Probability values.
			actual[i].Probability = 0
		}
		require.Equal(t, tc.expected, actual)
	}
}
