package inspect

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
)

type AdmissibleCodeTC struct {
	path     string
	code     string
	cursor   string
	language lang.Language
	expected bool
}

func TestAdmissibleCode(t *testing.T) {
	tcs := []AdmissibleCodeTC{
		// basic example (admissible)
		AdmissibleCodeTC{
			path: "test.js",
			code: `
				function computeSum(nums) {
					// add all the nums
					let $sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
			`,
			cursor:   "$",
			language: lang.JavaScript,
			expected: true,
		},
		// multiple cursors
		AdmissibleCodeTC{
			path: "test.js",
			code: `
				function computeSum(nums) {
					// add all the nums
					let $sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += n$ums[i]
					}
					return sum
				}
			`,
			cursor:   "$",
			language: lang.JavaScript,
			expected: false,
		},
		// no cursor
		AdmissibleCodeTC{
			path: "test.js",
			code: `
				function computeSum(nums) {
					// add all the nums
					let sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
			`,
			cursor:   "$",
			language: lang.JavaScript,
			expected: false,
		},
		// cursor in comment:
		AdmissibleCodeTC{
			path: "test.js",
			code: `
				function computeSum(nums) {
					// add $all the nums
					let sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
			`,
			cursor:   "$",
			language: lang.JavaScript,
			expected: false,
		},
		// no letters in cursor line:
		AdmissibleCodeTC{
			path: "test.js",
			code: `
				function computeSum(nums) {
					// add all the nums
					let sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}$
					return sum
				}
			`,
			cursor:   "$",
			language: lang.JavaScript,
			expected: false,
		},
		// encoded context vector is empty:
		AdmissibleCodeTC{
			path: "test.js",
			code: `
				// add all the $nums
				function computeSum(nums) {
					let sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
			`,
			cursor:   "$",
			language: lang.JavaScript,
			expected: false,
		},
		// cursor in string:
		AdmissibleCodeTC{
			path: "test.js",
			code: `
				// add all the nums
				function computeSum(nums) {
					let x = "hell$o"
					let sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
			`,
			cursor:   "$",
			language: lang.JavaScript,
			expected: false,
		},
		// cursor inside reg exp literal
		AdmissibleCodeTC{
			path: "test.js",
			code: `
				var pattern = /[0-9a$-zA-Z]+/g;
			`,
			cursor:   "$",
			language: lang.JavaScript,
			expected: true,
		},
		// cursor after division (admissible)
		AdmissibleCodeTC{
			path: "test.js",
			code: `
				var pattern = 1/9$2
			`,
			cursor:   "$",
			language: lang.JavaScript,
			expected: true,
		},
	}
	for i, tc := range tcs {
		actual := admissibleCode(tc.path, tc.code, tc.cursor, lexicalv0.NewLangGroup(tc.language))
		if actual != tc.expected {
			t.Errorf(
				"test case %d failed: actual: %t, expected: %t",
				i, actual, tc.expected,
			)
		}
	}
}

type CursorLineTC struct {
	code     string
	cursor   string
	expected string
}

func TestCursorLine(t *testing.T) {
	tcs := []CursorLineTC{
		CursorLineTC{
			code: `
				function computeSum(nums) {
					// add all the nums
					let $sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
			`,
			cursor: "$",
			expected: "					let sum = 0",
		},
		CursorLineTC{
			code: `
				function computeSum(nums) {
					// add all the nums
					let sum = 0
					$for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
			`,
			cursor: "$",
			expected: "					for (i = 0; i < nums.length; i++) {",
		},
	}
	for i, tc := range tcs {
		actual := cursorLine(tc.code, tc.cursor)
		if actual != tc.expected {
			t.Errorf(
				"test case %d failed: actual: %s, expected: %s",
				i, actual, tc.expected,
			)
		}
	}
}

type ContainsLetterTC struct {
	s        string
	expected bool
}

func TestContainsLetter(t *testing.T) {
	tcs := []ContainsLetterTC{
		ContainsLetterTC{
			s:        "!@#$%^&*(){:><?` ,./';|",
			expected: false,
		},
		ContainsLetterTC{
			s:        "!@$%^x*)&",
			expected: true,
		},
		ContainsLetterTC{
			s:        "!@$%^X*)&",
			expected: true,
		},
	}
	for i, tc := range tcs {
		actual := containsLetter(tc.s)
		if actual != tc.expected {
			t.Errorf(
				"test case %d failed: actual: %t, expected %t",
				i, actual, tc.expected,
			)
		}
	}
}

type CursorInStringTC struct {
	code     string
	cursor   string
	expected bool
}

func TestCursorInString(t *testing.T) {
	tcs := []CursorInStringTC{
		CursorInStringTC{
			code:     `let x = "hell$o"`,
			cursor:   "$",
			expected: true,
		},
		CursorInStringTC{
			code: strings.Join(
				[]string{
					"let x = `",
					"hello$ world",
					"`",
				},
				"\n",
			),
			cursor:   "$",
			expected: true,
		},
		CursorInStringTC{
			code:     `let x = "hello"$`,
			cursor:   "$",
			expected: false,
		},
		CursorInStringTC{
			code: strings.Join(
				[]string{
					"let x = `",
					"hello world",
					"`$",
				},
				"\n",
			),
			cursor:   "$",
			expected: false,
		},
	}
	for i, tc := range tcs {
		actual := cursorInString(tc.code, tc.cursor)
		if actual != tc.expected {
			t.Errorf(
				"test case %d failed: actual: %t, expected %t",
				i, actual, tc.expected,
			)
		}
	}
}
