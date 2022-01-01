package text

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/assert"
)

type cursorLineTC struct {
	Desc     string
	Code     string
	Cursor   string
	Expected string
	Pos      int
}

func requireSelectedBuffer(code, cursor string) data.SelectedBuffer {
	parts := strings.Split(code, cursor)
	switch len(parts) {
	case 1:
		return data.NewBuffer(parts[0]).Select(data.Selection{Begin: len(parts[0]), End: len(parts[0])})
	case 2:
		return data.NewBuffer(parts[0] + parts[1]).Select(data.Selection{Begin: len(parts[0]), End: len(parts[0])})
	default:
		panic(fmt.Sprintf("code must contain 1 or 2 parts, got %d", len(parts)))
	}
}

func TestCursorLine(t *testing.T) {
	tcs := []cursorLineTC{
		{
			Desc:     "no ending line",
			Code:     "\nfoo$ bar",
			Cursor:   "$",
			Expected: "foo bar",
			Pos:      3,
		},
		{
			Desc:     "no starting line",
			Code:     "foo$ bar\n",
			Cursor:   "$",
			Expected: "foo bar",
			Pos:      3,
		},
		{
			Desc:     "no newlines",
			Code:     "foo $bar",
			Cursor:   "$",
			Expected: "foo bar",
			Pos:      4,
		},
		{
			Desc: "middle of line",
			Code: `
				function computeSum(nums) {
					// add all the nums
					let s$um = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
			`,
			Cursor: "$",
			Expected: "					let sum = 0",
			Pos: 10,
		},
		{
			Desc: "start of line",
			Code: `
				function computeSum(nums) {
					// add all the nums
					let sum = 0
					$for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
			`,
			Cursor: "$",
			Expected: "					for (i = 0; i < nums.length; i++) {",
			Pos: 5,
		},
		{
			Desc: "end of line",
			Code: `
				function computeSum(nums) {
					// add all the nums
					let sum = 0
					for (i = 0; i < nums.length; i++) {$
						sum += nums[i]
					}
					return sum
				}
			`,
			Cursor: "$",
			Expected: "					for (i = 0; i < nums.length; i++) {",
			Pos: 40,
		},
	}
	for i, tc := range tcs {
		sb := requireSelectedBuffer(tc.Code, tc.Cursor)
		actualLine, actualPos := cursorLine(sb)
		assert.Equal(t, tc.Expected, actualLine, "test case %d: %s", i, tc.Desc)
		assert.Equal(t, tc.Pos, actualPos, "test case %d: %s", i, tc.Desc)
	}
}

type cursorInCommentTC struct {
	Desc     string
	Path     string
	Code     string
	Cursor   string
	Expected bool
}

func TestCursorInComment(t *testing.T) {
	tcs := []cursorInCommentTC{
		{
			Desc:     "js file, cursor on same line as comment, before comment",
			Path:     "test.js",
			Code:     "foo($) // some comment",
			Cursor:   "$",
			Expected: false,
		},
		{
			Desc:     "js file, cursor on same line as comment, after comment",
			Path:     "test.js",
			Code:     "foo() // some$ comment",
			Cursor:   "$",
			Expected: true,
		},
		{
			Desc: "js file, after single line comment",
			Path: "test.js",
			Code: `
				function computeSum(nums) {
					// add all the nums
					let $sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
			`,
			Cursor:   "$",
			Expected: false,
		},
		{
			Desc: "js file in single line comment",
			Path: "test.js",
			Code: `
				function computeSum(nums) {
					// add $all the nums
					let sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
			`,
			Cursor:   "$",
			Expected: true,
		},
		{
			Desc: "js file inside multi line comment",
			Path: "test.js",
			Code: `
				/*
				function computeSum(nums) {
					// add all the nums
					let $sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
				*/
			`,
			Cursor:   "$",
			Expected: true,
		},
		{
			Desc: "js file outside multi line comment",
			Path: "test.js",
			Code: `
				/*
				Note:
				*/
				function computeSum(nums) {
					// add all the nums
					let $sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
			`,
			Cursor:   "$",
			Expected: false,
		},
		{
			Desc: "js file, multiple multi line comments",
			Path: "test.js",
			Code: `
				/*
				Note:
				*/
				const depth = 3
				/*
				function computeSum(nums) {
					// add all the nums
					let $sum = 0
					for (i = 0; i < nums.length; i++) {
						sum += nums[i]
					}
					return sum
				}
				*/
			`,
			Cursor:   "$",
			Expected: true,
		},
		{
			Desc: "python file, multiple multi line comments",
			Path: "test.py",
			Code: `
				'''
				NOTE: 
				'''
				depth = 3
				'''
				def computeSum(nums):
					# add all the nums
					$sum = 0
					for i in range(len(nums)):
						sum = sum + nums[i]
					return sum
				'''
			`,
			Cursor:   "$",
			Expected: true,
		},
		{
			Desc: "python file, check multiline outside comment",
			Path: "test.py",
			Code: `
				'''
				NOTE: 
				'''
				depth = 3
				def computeSum(nums):
					# add all the nums
					$sum = 0
					for i in range(len(nums)):
						sum = sum + nums[i]
					return sum
			`,
			Cursor:   "$",
			Expected: false,
		},
	}
	for i, tc := range tcs {
		sb := requireSelectedBuffer(tc.Code, tc.Cursor)

		actual := CursorInComment(sb, lang.FromFilename(tc.Path))

		assert.Equal(t, tc.Expected, actual, "test case %d: %s", i, tc.Desc)
	}
}
