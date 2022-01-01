package inspect

import (
	"math"
	"testing"
)

type normalizeTC struct {
	weights            []float32
	expectedNormalized []float32
	expectedError      bool
}

func TestNormalize(t *testing.T) {
	tcs := []normalizeTC{
		normalizeTC{
			weights:            []float32{3.4, 10.1, 16.5, 70},
			expectedNormalized: []float32{0.034, 0.101, 0.165, 0.7},
			expectedError:      false,
		},
		normalizeTC{
			weights:            []float32{3.4, 10.1, 16.5, -70},
			expectedNormalized: nil,
			expectedError:      true,
		},
	}
	for i, tc := range tcs {
		actual, err := normalize(tc.weights)
		actualError := err != nil
		if actualError != tc.expectedError {
			t.Errorf(
				"test case %d failed: actual error: %t, expected error: %t",
				i, actualError, tc.expectedError,
			)
		}
		if len(actual) != len(tc.expectedNormalized) {
			t.Errorf(
				"test case %d failed: actual length: %d, expected length: %d",
				i, len(actual), len(tc.expectedNormalized),
			)
			return
		}
		for j := range actual {
			if math.Abs(float64(actual[j]-tc.expectedNormalized[j])) >= 1e-6 {
				t.Errorf(
					"test case %d failed at position %d: actual: %f, expected: %f",
					i, j, actual[j], tc.expectedNormalized[j],
				)
			}
		}
	}
}

type aggregateTC struct {
	values         [][]Attention
	expectedValues Attention
	expectedError  bool
}

func TestAggregate(t *testing.T) {
	tcs := []aggregateTC{
		aggregateTC{
			values: [][]Attention{
				[]Attention{
					Attention{
						[]float32{3, 8, 1},
						[]float32{7, 1, 2},
					},
					Attention{
						[]float32{8, 1, 3},
						[]float32{7, 4, 2},
					},
				},
				[]Attention{
					Attention{
						[]float32{8, 3, 8},
						[]float32{12, 5, 3},
					},
					Attention{
						[]float32{8, 1, 9},
						[]float32{5, 9, 2},
					},
				},
			},
			expectedValues: Attention{
				[]float32{27.0 / 61, 13.0 / 61, 21.0 / 61},
				[]float32{31.0 / 59, 19.0 / 59, 9.0 / 59},
			},
			expectedError: false,
		},
		aggregateTC{
			values: [][]Attention{
				[]Attention{
					Attention{
						[]float32{3, 8, 1},
						[]float32{7, 1, 2},
					},
					Attention{
						[]float32{8, 1, 3},
						[]float32{7, 4, 2},
					},
				},
				[]Attention{
					Attention{
						[]float32{8, 3, 8},
						[]float32{12, 5, 3},
					},
					Attention{
						[]float32{8, 1, 9},
						[]float32{5, 9, 2, 4},
					},
				},
			},
			expectedValues: nil,
			expectedError:  true,
		},
	}
	for i, tc := range tcs {
		actual, err := aggregate(tc.values)
		actualError := err != nil
		if actualError != tc.expectedError {
			t.Errorf(
				"test case %d failed: actual error: %t, expected error: %t",
				i, actualError, tc.expectedError,
			)
		}
		if len(actual) != len(tc.expectedValues) {
			t.Errorf(
				"test case %d failed: actual num rows: %d, expected num rows: %d",
				i, len(actual), len(tc.expectedValues),
			)
			return
		}
		for j := range actual {
			if len(actual[j]) != len(tc.expectedValues[j]) {
				t.Errorf(
					"test case %d failed at row %d: actual length: %d, expected length: %d",
					i, j, len(actual[j]), len(tc.expectedValues[j]),
				)
				return
			}
			for k := range actual[j] {
				if math.Abs(float64(actual[j][k]-tc.expectedValues[j][k])) >= 1e-6 {
					t.Errorf(
						"test case %d failed at position (%d, %d): actual: %f, expected: %f",
						i, j, k, actual[j][k], tc.expectedValues[j][k],
					)
				}
			}
		}
	}
}
