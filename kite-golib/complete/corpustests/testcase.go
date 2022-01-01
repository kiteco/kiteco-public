package corpustests

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/driver"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Defining test state types
const (
	OkState   = "ok"
	FailState = "fail"
	SlowState = "slow"
)

var backQuoteRegExp = regexp.MustCompile("`[^`]*`")

// TestCase encapsulates the information needed to run a corpus test
type TestCase struct {
	Name string

	Exact    bool
	Expected []ExpectedCompletion
	Insert   string

	Status   string
	SB       data.SelectedBuffer
	Filename string
}

// ExpectedCompletion contains the expected completion information
type ExpectedCompletion struct {
	Insert  string
	Display string
	Hint    string
	Rank    int
	Not     bool
	Any     bool
}

// String formats the expected completion for printing
func (ec ExpectedCompletion) String() string {
	if ec.Not {
		return fmt.Sprintf("@! %s", ec.Insert)
	}
	if ec.Any {
		return fmt.Sprintf("@. insert: %s display: %s hint: %s", ec.Insert, ec.Display, ec.Hint)
	}
	if ec.Display == "" {
		return fmt.Sprintf("@%d insert: %s", ec.Rank, ec.Insert)
	}
	return fmt.Sprintf("@%d insert: %s display: %s", ec.Rank, ec.Insert, ec.Display)
}

// RequireFunc tests that the condition is met, otherwise it outputs an error
type RequireFunc func(cond bool, errStr string, args ...interface{})

// CreateTestCase returns a test case given the input test case description lines
func CreateTestCase(t *testing.T, name, filename, cursor string, lines []string, require RequireFunc) TestCase {
	var status, insert string
	var exact bool
	var expected []ExpectedCompletion
	for _, line := range lines {
		switch {
		case strings.Contains(line, cursor):
			insert = strings.TrimSpace(line)
		case strings.Contains(line, "@"):
			// ['@rank', 'expected insert' | '...', 'expected display'?, 'expected hint'?] | ['@EXACT'] | ['@!', 'expected insert'] | ['@.', 'expected insert']

			parts := getParts(line)
			require(len(parts) > 0 && len(parts) < 5, "expected 1-4 fields for '%s', got '%d'", line, len(parts))

			if len(parts) == 1 {
				require(parts[0] == "@EXACT", "expected '@EXACT', got '%s'", parts[0])
				exact = true
				continue
			}

			ec := ExpectedCompletion{}

			if parts[1] != "..." {
				ec.Insert = parts[1]
			}
			if len(parts) > 2 {
				ec.Display = parts[2]
			}
			if len(parts) > 3 {
				ec.Hint = parts[3]
			}

			i := strings.Index(parts[0], "@")
			require(i >= 0, "unable to find '@' in '%s', line '%s'", parts[0], line)

			remaining := parts[0][i+1:]
			switch remaining {
			case "!":
				ec.Not = true
			case ".":
				ec.Any = true
			default:
				r, err := strconv.ParseInt(remaining, 10, 64)
				require(err == nil, "unable to parse rank from '%s': %v", parts[0], err)
				ec.Rank = int(r)
			}

			expected = append(expected, ec)
		case strings.Contains(line, "status"):
			parts := strings.Fields(line) // ['status:', 'ok/slow/fail']
			require(len(parts) == 2, "expected 2 fields for '%s', got %d", line, len(parts))

			status = strings.ToLower(parts[1])
			switch status {
			case OkState, SlowState, FailState:
			default:
				t.Errorf("unsupported status '%s' in line '%s'", status, line)
			}
		}
	}

	require(
		len(expected) > 0 || exact,
		"need to provide either expected completions or the exact flag",
	)

	require(status != "", "unable to find status")
	require(insert != "", "unable to find insert")

	return TestCase{
		Name:     name,
		Insert:   insert,
		Exact:    exact,
		Expected: expected,
		Status:   status,
		Filename: path.Join("/", filename),
	}
}

func getParts(s string) []string {
	var blocks []string
	placeholder := "##STRING##"
	replacer := func(m string) string {
		blocks = append(blocks, m)
		return placeholder
	}
	s = backQuoteRegExp.ReplaceAllStringFunc(s, replacer)
	parts := strings.Fields(s)
	var counter uint
	if len(blocks) > 0 {
		for i := range parts {
			if parts[i] == placeholder {
				initialString := blocks[counter]
				parts[i] = initialString[1 : len(initialString)-1]
				counter++
			}
		}
	}
	return parts
}

// RunTestCase compares the completions returned to the expected completions
func RunTestCase(ctx kitectx.Context, t *testing.T, c TestCase, compls []data.RCompletion) {
	assert := func(c TestCase, cond bool, fstr string, args ...interface{}) bool {
		if !cond {
			errorTestCase(t, c)
			t.Errorf("\n"+fstr, args...)
			return false
		}
		return true
	}

	for _, ec := range c.Expected {

		displays := make(map[string]int)
		for curr, comp := range compls {
			prev, duplicate := displays[comp.Display]
			assert(c, !duplicate, "duplicate displays at %d and %d, got:\n%s", prev, curr, printCompletions(compls))
			if duplicate {
				break
			}
			displays[comp.Display] = curr
		}

		if ec.Not {
			var any bool
			for i, comp := range compls {
				ok := assert(c,
					(ec.Insert != "" && comp.Snippet.Text != ec.Insert) || (ec.Display != "" && ec.Display != comp.Display),
					"cannot have %s at any rank, found at %d",
					ec.Insert, i,
				)
				if !ok {
					any = true
				}
			}

			if any {
				t.Errorf("\nCompletions:\n%s", printCompletions(compls))
			}

			continue
		}

		if ec.Any {
			var exists bool
			for _, comp := range compls {
				if ec.Insert != "" && comp.Snippet.Text != ec.Insert {
					continue
				}
				if ec.Display != "" && comp.Display != ec.Display {
					continue
				}
				if ec.Hint != "" && comp.Hint != ec.Hint {
					continue
				}
				exists = true
				break
			}
			assert(c, exists, "could not find { %s }, got:\n%s", ec.String(), printCompletions(compls))
			continue
		}

		ok := assert(c,
			len(compls) > ec.Rank,
			"expected at least %d completions, got %d:\n%s",
			ec.Rank+1, len(compls), printCompletions(compls),
		)

		if !ok {
			continue
		}

		insert := compls[ec.Rank].Completion.Snippet.Text
		display := compls[ec.Rank].Display
		hint := compls[ec.Rank].Hint

		if ec.Insert != "" {
			assert(c,
				ec.Insert == insert,
				"@rank %d, expected: %s got:\n%s",
				ec.Rank, ec.Insert, printCompletions(compls),
			)
		}
		if ec.Display != "" {
			assert(c,
				ec.Display == display,
				"@rank %d, expected display text: %s got: %s:\n%s",
				ec.Rank, ec.Display, display, printCompletions(compls),
			)
		}
		if ec.Hint != "" {
			assert(c,
				ec.Hint == hint,
				"@rank %d, expected hint text: %s got: %s:\n%s",
				ec.Rank, ec.Hint, hint, printCompletions(compls),
			)
		}
	}

	if c.Exact {
		assert(c,
			len(compls) == len(c.Expected),
			"expected exactly %d completions, got %d:\n%s",
			len(c.Expected), len(compls), printCompletions(compls),
		)
	}
}

const caseFmt = `
case %s:%s (%s):
input:
%s
exact: %v
expected:
%s
`

func errorTestCase(t *testing.T, c TestCase) {
	prefix := c.SB.TextAt(data.Selection{
		End: c.SB.Selection.Begin,
	})
	selected := c.SB.TextAt(c.SB.Selection)
	suffix := c.SB.TextAt(data.Selection{
		Begin: c.SB.Selection.End,
		End:   len(c.SB.Text()),
	})

	display := strings.Join([]string{
		prefix,
		"⦉",
		selected,
		"⦊",
		suffix,
	}, "")

	var ecs []string
	for _, e := range c.Expected {
		ecs = append(ecs, e.String())
	}

	t.Errorf("failed assert for case:")
	t.Errorf(caseFmt, c.Filename, c.Name, c.Status,
		display, c.Exact, strings.Join(ecs, "\n"))
}

func printCompletions(comps []data.RCompletion) string {
	var parts []string
	for i, c := range comps {
		var debugStr string
		switch debug := c.Debug.(type) {
		case driver.Completion:
			debugStr = fmt.Sprintf("%f", debug.Meta.Score)
		case string:
			debugStr = debug
		}
		parts = append(parts, fmt.Sprintf("%d: %s / %s / %s (%s)", i, c.Snippet.Text, c.Display, c.Hint, debugStr))
	}
	return strings.Join(parts, "\n")
}

func removeSpace(s string) string {
	return strings.Replace(s, " ", "", -1)
}
