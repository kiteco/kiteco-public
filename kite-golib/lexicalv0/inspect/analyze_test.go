package inspect

import "testing"

type equalTC struct {
	x        []int
	y        []int
	expected bool
}

func TestEqual(t *testing.T) {
	tcs := []equalTC{
		equalTC{
			x:        []int{-12, 38, 7},
			y:        []int{-12, 38, 7},
			expected: true,
		},
		equalTC{
			x:        []int{-12, 38, 7},
			y:        []int{-12, 38, 8},
			expected: false,
		},
		equalTC{
			x:        []int{-12, 38, 7},
			y:        []int{-12, 38, 7, 9},
			expected: false,
		},
		equalTC{
			x:        []int{-12, 38, 7},
			y:        []int{},
			expected: false,
		},
		equalTC{
			x:        []int{},
			y:        []int{},
			expected: true,
		},
	}
	for i, tc := range tcs {
		actual := equal(tc.x, tc.y)
		if tc.expected != actual {
			t.Errorf(
				"test case %d failed: actual %t, expected %t",
				i, actual, tc.expected,
			)
		}
	}
}

type findTC struct {
	haystack      []int
	needle        []int
	expectedIndex int
	expectedFound bool
}

func TestFind(t *testing.T) {
	tcs := []findTC{
		findTC{
			haystack:      []int{72, 48, 12, 68, 60, 12, 56},
			needle:        []int{12, 68, 60, 12},
			expectedIndex: 2,
			expectedFound: true,
		},
		findTC{
			haystack:      []int{72, 48, 12, 68, 60, 32, 56},
			needle:        []int{12, 68, 60, 12},
			expectedIndex: -1,
			expectedFound: false,
		},
		findTC{
			haystack:      []int{72, 48, 12, 68, 60, 32, 56},
			needle:        []int{12, 68, 60, 32, 56, 91},
			expectedIndex: -1,
			expectedFound: false,
		},
		findTC{
			haystack:      []int{72, 48, 12, 68, 60, 32, 56},
			needle:        []int{68, 60, 32, 56},
			expectedIndex: 3,
			expectedFound: true,
		},
	}
	for i, tc := range tcs {
		actualIndex, actualFound := find(tc.haystack, tc.needle)
		if tc.expectedFound != actualFound {
			t.Errorf(
				"test case %d failed: actual found: %t, expected found: %t",
				i, actualFound, tc.expectedFound,
			)
		}
		if tc.expectedIndex != actualIndex {
			t.Errorf(
				"test case %d failed: actual index: %d, expected index: %d",
				i, actualIndex, tc.expectedIndex,
			)
		}
	}
}
