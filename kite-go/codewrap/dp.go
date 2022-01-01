package codewrap

import (
	"strings"
	"unicode"
)

func leadingIndent(line string, tabwidth int) int {
	var n int
	for _, rune := range line {
		if rune == '\t' {
			n += tabwidth
		} else if unicode.IsSpace(rune) {
			n++
		} else {
			break
		}
	}
	return n
}

func leadingIndentStr(line string) string {
	var s string
	for _, rune := range line {
		if unicode.IsSpace(rune) {
			s += string(rune)
		} else {
			break
		}
	}
	return s
}

func escaped(s string) string {
	if s == "\n" {
		return "\\n"
	} else if s == "\t" {
		return "\\t"
	} else {
		return s
	}
}

func width(s string, tabWidth int) int {
	var n int
	for _, rune := range s {
		if rune == '\t' {
			n += tabWidth
		} else {
			n++
		}
	}
	return n
}

// subProblem represents a layout sub-problem solved in the dynamic program.
type subProblem struct {
	index  int // index of the next token to be laid out
	column int // ending column of the previous token
}

// solution represents a solution to a layout sub-problem.
type solution struct {
	layout []string
	cost   float64
}

// layoutDP represents a dynamic program for solving the code layout problem.
type layoutDP struct {
	cache  map[subProblem]solution
	tokens []Token
	opts   Options
}

// newLayoutDP creates a new dynamic program for laying out the specified tokens
// with a given column limit.
func newLayoutDP(tokens []Token, opts Options) layoutDP {
	return layoutDP{
		cache:  make(map[subProblem]solution),
		tokens: tokens,
		opts:   opts,
	}
}

// Look up a solution to a DP subproblem in the cache, or compute the solution
// if it's not present.
func (dp *layoutDP) solve(index, column, indent int) solution {
	key := subProblem{index, column}

	// Try to look up a cached solution
	if solution, found := dp.cache[key]; found {
		return solution
	}

	// Otherwise, compute the solution for this subproblem and cache it
	solution := dp.solveImpl(index, column, indent)
	dp.cache[key] = solution
	return solution
}

// Solve a DP subproblem
func (dp *layoutDP) solveImpl(index, column, indent int) solution {
	// Base case: at the end of the line
	if index == len(dp.tokens) {
		return solution{[]string{""}, 0.}
	}

	s := dp.tokens[index].String()

	// Find the last non-comment token
	lastToken := dp.tokens[len(dp.tokens)-1]
	if lastToken.IsComment() && len(dp.tokens) > 1 {
		lastToken = dp.tokens[len(dp.tokens)-2]
	}

	// Compute cost for continuing on the same line
	if column > indent && dp.tokens[index-1].InsertSpace(dp.tokens[index]) {
		s = " " + s
	}

	var penalty float64
	if column+width(s, dp.opts.TabWidth) > dp.opts.Columns {
		penalty = float64(ExtraColumnCost * (column + width(s, dp.opts.TabWidth) - dp.opts.Columns))
	}

	subSolution := dp.solve(index+1, column+len(s), indent)

	// Construct a solution from the sub-solution
	var best solution
	best.layout = append([]string{}, subSolution.layout...)
	best.layout[0] = s + best.layout[0]
	best.cost = subSolution.cost + penalty

	// Compute cost for splitting at this token
	if index > 0 {
		transition, continuation := dp.tokens[index-1].SplitCost(dp.tokens[index])
		if transition >= 0. {
			extra := dp.opts.TabWidth
			prefix := strings.Repeat(" ", indent+extra) + dp.tokens[index].String()
			subSolution := dp.solve(index+1, len(prefix), indent)
			// It could be that it is impossible to lay out this code from this column
			if subSolution.cost+transition < best.cost {
				// Construct a solution from the sub-solution
				best.layout = append([]string{continuation}, subSolution.layout...)
				best.layout[1] = prefix + best.layout[1]
				best.cost = subSolution.cost + transition
			}
		}
	}

	return best
}

// Layout a list of tokens on one or more lines so as not to exceed the given column limit.
// This version uses a dynamic program to minimize a sum of splitting costs.
func layoutLine(tokens []Token, raw string, opts Options) []string {
	// First count the number of leading tabs
	indent := leadingIndent(raw, opts.TabWidth)

	// If the line is empty then return a single empty line as output
	if len(tokens) == 0 {
		return []string{""}
	}

	// Initialize DP
	dp := newLayoutDP(tokens, opts)
	solution := dp.solve(0, indent, indent)
	solution.layout[0] = strings.Repeat(" ", indent) + solution.layout[0]
	return solution.layout
}
