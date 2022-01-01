package validation

import (
	"errors"
	"math"
	"testing"

	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

type countTC struct {
	relevant  []string
	retrieved []string
	expected  Stats
	threshold float64
}

func approx(x, y Stats, threshold float64) bool {
	if math.Abs(x.Precision-y.Precision) > threshold {
		return false
	}
	if math.Abs(x.Recall-y.Recall) > threshold {
		return false
	}
	if math.Abs(x.F1-y.F1) > threshold {
		return false
	}
	return true
}

func TestCount(t *testing.T) {
	tcs := []countTC{
		countTC{
			relevant:  []string{"alpha", "beta", "gamma", "delta"},
			retrieved: []string{"beta", "gamma", "epsilon", "zeta", "eta"},
			expected: Stats{
				Precision: 2.0 / 5,
				Recall:    2.0 / 4,
				F1:        2.0 / 4.5,
			},
			threshold: 0.001,
		},
	}
	for _, tc := range tcs {
		actual := Count(tc.relevant, tc.retrieved)
		require.True(t, approx(tc.expected, actual, tc.threshold))
	}
}

type mockRecommender struct {
	recommend       map[string][]string
	recommendBlocks map[string]map[string][]recommend.Block
}

var errMock = errors.New("mock error")

func (r mockRecommender) Recommend(ctx kitectx.Context, request recommend.Request) ([]recommend.File, error) {
	paths, ok := r.recommend[request.Location.CurrentPath]
	if !ok {
		return nil, errMock
	}
	var recs []recommend.File
	for _, path := range paths {
		recs = append(recs, recommend.File{Path: path})
	}
	return recs, nil
}

func (r mockRecommender) RecommendBlocks(ctx kitectx.Context, request recommend.BlockRequest) ([]recommend.File, error) {
	curr, ok := r.recommendBlocks[request.Location.CurrentPath]
	if !ok {
		return nil, errMock
	}
	var files []recommend.File
	for _, inspectFile := range request.InspectFiles {
		blocks, ok := curr[inspectFile.Path]
		if !ok {
			return nil, errMock
		}
		files = append(files, recommend.File{
			Path:   inspectFile.Path,
			Blocks: blocks,
		})
	}
	return files, nil
}

func (r mockRecommender) RankedFiles() ([]recommend.File, error) {
	return nil, nil
}

func (r mockRecommender) ShouldRebuild() (bool, error) {
	return false, nil
}

func meanOfMeans(chunks ...[]float64) float64 {
	mean := func(nums []float64) float64 {
		var sum float64
		for _, num := range nums {
			sum += num
		}
		return sum / float64(len(nums))
	}

	var means []float64
	for _, chunk := range chunks {
		means = append(means, mean(chunk))
	}
	return mean(means)
}

type measureFileTC struct {
	recommender     mockRecommender
	validationData  map[PullRequestID][]recommend.File
	threshold       float64
	expectedStats   Stats
	expectedRecords []Record
}

func TestMeasureFiles(t *testing.T) {
	tcs := []measureFileTC{
		measureFileTC{
			recommender: mockRecommender{
				recommend: map[string][]string{
					"alpha":   []string{"beta", "gamma"},
					"beta":    []string{"alpha", "delta"},
					"gamma":   []string{"alpha", "beta"},
					"delta":   []string{"alpha", "beta"},
					"epsilon": []string{"alpha", "delta"},
				},
			},
			validationData: map[PullRequestID][]recommend.File{
				"11": []recommend.File{
					recommend.File{Path: "alpha"},
					recommend.File{Path: "beta"},
				},
				"21": []recommend.File{
					recommend.File{Path: "alpha"},
					recommend.File{Path: "delta"},
					recommend.File{Path: "epsilon"},
				},
				"32": []recommend.File{
					recommend.File{Path: "epsilon"},
				},
			},
			threshold: 0.001,
			expectedStats: Stats{
				Precision: meanOfMeans(
					[]float64{1. / 2, 1. / 2},
					[]float64{0, 1. / 2, 1},
				),
				Recall: meanOfMeans(
					[]float64{1, 1},
					[]float64{0, 1. / 2, 1},
				),
				F1: meanOfMeans(
					[]float64{2. / 3, 2. / 3},
					[]float64{0, 1. / 2, 1},
				),
			},
			expectedRecords: []Record{
				Record{
					Base:        "alpha",
					Recommended: "beta",
					IsRelevant:  true,
				},
				Record{
					Base:        "alpha",
					Recommended: "gamma",
					IsRelevant:  false,
				},
				Record{
					Base:        "beta",
					Recommended: "alpha",
					IsRelevant:  true,
				},
				Record{
					Base:        "beta",
					Recommended: "delta",
					IsRelevant:  false,
				},
				Record{
					Base:        "alpha",
					Recommended: "beta",
					IsRelevant:  false,
				},
				Record{
					Base:        "alpha",
					Recommended: "gamma",
					IsRelevant:  false,
				},
				Record{
					Base:        "delta",
					Recommended: "alpha",
					IsRelevant:  true,
				},
				Record{
					Base:        "delta",
					Recommended: "beta",
					IsRelevant:  false,
				},
				Record{
					Base:        "epsilon",
					Recommended: "alpha",
					IsRelevant:  true,
				},
				Record{
					Base:        "epsilon",
					Recommended: "delta",
					IsRelevant:  true,
				},
			},
		},
		measureFileTC{
			recommender: mockRecommender{
				recommend: map[string][]string{
					"alpha":   []string{"beta", "gamma"},
					"beta":    []string{"alpha", "delta"},
					"gamma":   []string{"alpha", "beta"},
					"delta":   []string{"alpha", "beta"},
					"epsilon": []string{"alpha", "delta"},
				},
			},
			validationData: map[PullRequestID][]recommend.File{
				"11": []recommend.File{
					recommend.File{Path: "alpha"},
					recommend.File{Path: "beta"},
				},
				"21": []recommend.File{
					recommend.File{Path: "alpha"},
					recommend.File{Path: "delta"},
					recommend.File{Path: "epsilon"},
				},
				"32": []recommend.File{
					recommend.File{Path: "epsilon"},
					recommend.File{Path: "zeta"},
					recommend.File{Path: "eta"},
				},
			},
			expectedStats: Stats{
				Precision: meanOfMeans(
					[]float64{1. / 2, 1. / 2},
					[]float64{0, 1. / 2, 1},
					[]float64{0, 0},
				),
				Recall: meanOfMeans(
					[]float64{1, 1},
					[]float64{0, 1. / 2, 1},
					[]float64{0, 0},
				),
				F1: meanOfMeans(
					[]float64{2. / 3, 2. / 3},
					[]float64{0, 1. / 2, 1},
					[]float64{0, 0},
				),
			},
			threshold: 0.001,
			expectedRecords: []Record{
				Record{
					Base:        "alpha",
					Recommended: "beta",
					IsRelevant:  true,
				},
				Record{
					Base:        "alpha",
					Recommended: "gamma",
					IsRelevant:  false,
				},
				Record{
					Base:        "beta",
					Recommended: "alpha",
					IsRelevant:  true,
				},
				Record{
					Base:        "beta",
					Recommended: "delta",
					IsRelevant:  false,
				},
				Record{
					Base:        "alpha",
					Recommended: "beta",
					IsRelevant:  false,
				},
				Record{
					Base:        "alpha",
					Recommended: "gamma",
					IsRelevant:  false,
				},
				Record{
					Base:        "delta",
					Recommended: "alpha",
					IsRelevant:  true,
				},
				Record{
					Base:        "delta",
					Recommended: "beta",
					IsRelevant:  false,
				},
				Record{
					Base:        "epsilon",
					Recommended: "alpha",
					IsRelevant:  true,
				},
				Record{
					Base:        "epsilon",
					Recommended: "delta",
					IsRelevant:  true,
				},
				Record{
					Base:        "epsilon",
					Recommended: "alpha",
					IsRelevant:  false,
				},
				Record{
					Base:        "epsilon",
					Recommended: "delta",
					IsRelevant:  false,
				},
			},
		},
	}
	for _, tc := range tcs {
		validator := evaluator{
			recommender:    tc.recommender,
			validationData: tc.validationData,
		}
		stats, records := validator.evaluateFiles()
		require.True(t, approx(tc.expectedStats, stats, tc.threshold))
		require.ElementsMatch(t, tc.expectedRecords, records)
	}
}

type measureLineTC struct {
	recommender    mockRecommender
	validationData map[PullRequestID][]recommend.File
	expected       Stats
	expectedError  error
	threshold      float64
}

func TestMeasureLines(t *testing.T) {
	block := func(firstLine, lastLine int) recommend.Block {
		return recommend.Block{
			FirstLine: firstLine,
			LastLine:  lastLine,
		}
	}
	tcs := []measureLineTC{
		measureLineTC{
			recommender: mockRecommender{},
			validationData: map[PullRequestID][]recommend.File{
				"20": []recommend.File{
					recommend.File{Path: "alpha"},
					recommend.File{Path: "beta"},
				},
			},
			expectedError: errMock,
		},
		measureLineTC{
			recommender: mockRecommender{
				recommendBlocks: map[string]map[string][]recommend.Block{
					"alpha": map[string][]recommend.Block{
						"beta":  []recommend.Block{block(70, 91), block(107, 170)},
						"gamma": []recommend.Block{block(10, 10), block(30, 40)},
					},
					"beta": map[string][]recommend.Block{
						"alpha": []recommend.Block{block(40, 44)},
						"gamma": nil,
					},
					"gamma": map[string][]recommend.Block{
						"alpha": []recommend.Block{block(20, 80), block(100, 105)},
						"beta":  []recommend.Block{block(10, 11), block(20, 20)},
					},
				},
			},
			validationData: map[PullRequestID][]recommend.File{
				"11": []recommend.File{
					recommend.File{
						Path:   "alpha",
						Blocks: []recommend.Block{block(40, 50), block(81, 81)},
					},
					recommend.File{
						Path:   "beta",
						Blocks: []recommend.Block{block(20, 23)},
					},
				},
				"21": []recommend.File{
					recommend.File{
						Path:   "alpha",
						Blocks: []recommend.Block{block(10, 10), block(20, 25)},
					},
					recommend.File{
						Path:   "gamma",
						Blocks: []recommend.Block{block(11, 17)},
					},
				},
				"32": []recommend.File{
					recommend.File{
						Path:   "alpha",
						Blocks: []recommend.Block{block(30, 70), block(85, 93)},
					},
					recommend.File{
						Path:   "beta",
						Blocks: []recommend.Block{block(100, 111)},
					},
					recommend.File{
						Path:   "gamma",
						Blocks: []recommend.Block{block(1, 22), block(30, 30)},
					},
				},
			},
			expected: Stats{
				Precision: meanOfMeans(
					[]float64{0, 1},
					[]float64{0, 6. / 67},
					[]float64{5. / 86, 2. / 12, 0, 1, 41. / 67, 0},
				),
				Recall: meanOfMeans(
					[]float64{0, 5. / 12},
					[]float64{0, 6. / 7},
					[]float64{5. / 12, 2. / 23, 0, 1. / 10, 41. / 50, 0},
				),
				F1: meanOfMeans(
					[]float64{0, 10. / 17},
					[]float64{0, 6. / 37},
					[]float64{5. / 49, 4. / 35, 0, 2. / 11, 82. / 117, 0},
				),
			},
			threshold: 0.001,
		},
		measureLineTC{
			recommender: mockRecommender{
				recommendBlocks: map[string]map[string][]recommend.Block{
					"alpha": map[string][]recommend.Block{
						"beta":  []recommend.Block{block(70, 91), block(107, 170)},
						"gamma": []recommend.Block{block(10, 33), block(30, 40)},
					},
					"beta": map[string][]recommend.Block{
						"alpha": []recommend.Block{block(40, 44)},
						"gamma": nil,
					},
					"gamma": map[string][]recommend.Block{
						"alpha": []recommend.Block{block(20, 80), block(100, 105)},
						"beta":  []recommend.Block{block(10, 11), block(20, 20)},
					},
				},
			},
			validationData: map[PullRequestID][]recommend.File{
				"11": []recommend.File{
					recommend.File{
						Path:   "alpha",
						Blocks: []recommend.Block{block(40, 50), block(81, 81)},
					},
					recommend.File{
						Path:   "beta",
						Blocks: []recommend.Block{block(20, 23)},
					},
				},
				"21": []recommend.File{
					recommend.File{
						Path:   "alpha",
						Blocks: []recommend.Block{block(10, 10), block(20, 25)},
					},
					recommend.File{
						Path:   "gamma",
						Blocks: []recommend.Block{block(11, 17)},
					},
				},
				"32": []recommend.File{
					recommend.File{
						Path:   "alpha",
						Blocks: []recommend.Block{block(30, 70), block(85, 93)},
					},
					recommend.File{
						Path:   "beta",
						Blocks: []recommend.Block{block(100, 111)},
					},
					recommend.File{
						Path:   "gamma",
						Blocks: []recommend.Block{block(1, 22), block(30, 30)},
					},
				},
			},
			expectedError: errOverlappingBlocks,
		},
	}
	for _, tc := range tcs {
		validator := evaluator{
			recommender:    tc.recommender,
			validationData: tc.validationData,
		}
		actual, err := validator.evaluateLines()
		require.Equal(t, tc.expectedError, err)
		require.True(t, approx(tc.expected, actual, tc.threshold))
	}
}
