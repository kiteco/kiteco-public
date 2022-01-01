package argspec

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/errors"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/internal/testparser"
	"github.com/stretchr/testify/require"
)

// for tests to be isolated from changes to the defaultMaxLines.
const testMaxLines = 3

// NOTE(mna): implementing here as I only have one file (the ArgSpec+Arg)
// of the pythonimports package, but maybe this already exists in the
// package.
func printArgSpec(t *testing.T, spec *pythonimports.ArgSpec) string {
	var buf bytes.Buffer

	for _, arg := range spec.Args {
		fmt.Fprintf(&buf, "%s %q\n", arg.Name, arg.DefaultValue)
	}
	if spec.Vararg != "" {
		fmt.Fprintf(&buf, "Vararg: %s\n", spec.Vararg)
	}
	if spec.Kwarg != "" {
		fmt.Fprintf(&buf, "Kwarg: %s\n", spec.Kwarg)
	}
	return buf.String()
}

func assertParse(t *testing.T, expected string, src string, opts ...Option) {
	assertParseWithError(t, expected, src, errors.Unknown, opts...)
}

func assertParseWithError(t *testing.T, expected string, src string, expectedReason errors.Reason, opts ...Option) {
	opts = append([]Option{MaxLines(testMaxLines)}, opts...)
	spec, err := Parse([]byte(src), opts...)

	if expectedReason != errors.Unknown {
		require.Error(t, err)
		require.Equal(t, errors.ErrorReason(err), expectedReason, "%v", err)
		require.Nil(t, spec)
		return
	}
	require.NoError(t, err)
	require.NotNil(t, spec)

	got := printArgSpec(t, spec)
	// ignore leading/trailing whitespace
	expected = strings.TrimSpace(expected)
	got = strings.TrimSpace(got)
	require.Equal(t, expected, got)
}

func TestNoArgument(t *testing.T) {
	src := "get_state()"
	expected := ""
	assertParse(t, expected, src)
}

func TestOneArgNoValue(t *testing.T) {
	src := "set_state(state)"
	expected := `state ""`
	assertParse(t, expected, src)
}

func TestOneArgWithValue(t *testing.T) {
	src := "seed(seed=None)"
	expected := `seed "None"`
	assertParse(t, expected, src)
}

func TestRandInt(t *testing.T) {
	src := "randint(low, high=None, size=None, dtype='l')"
	expected := `
low ""
high "None"
size "None"
dtype "'l'"
`
	assertParse(t, expected, src)
}

func TestChoice(t *testing.T) {
	src := "choice(a, size=None, replace=True, p=None)"
	expected := `
a ""
size "None"
replace "True"
p "None"
`
	assertParse(t, expected, src)
}

func TestUniform(t *testing.T) {
	src := "uniform(low=0.0, high=1.0, size=None)"
	expected := `
low "0.0"
high "1.0"
size "None"
`
	assertParse(t, expected, src)
}

func TestRand(t *testing.T) {
	src := "rand(d0, d1, ..., dn)"
	expected := `
d0 ""
d1 ""
... ""
dn ""
`
	assertParse(t, expected, src)
}

func TestLognormal(t *testing.T) {
	src := "lognormal(mean=0.0, sigma=1.0, size=None)"
	expected := `
mean "0.0"
sigma "1.0"
size "None"
`
	assertParse(t, expected, src)
}

func TestMultivariateNormal(t *testing.T) {
	src := "multivariate_normal(mean, cov[, size, check_valid, tol])"
	expected := `
mean ""
cov ""
size "..."
check_valid "..."
tol "..."
`
	assertParse(t, expected, src)
}

func TestVararg(t *testing.T) {
	src := "fn(a, *b)"
	expected := `
a ""
Vararg: b
`
	assertParse(t, expected, src)
}

func TestVarargFirst(t *testing.T) {
	src := "fn(*a)"
	expected := `
Vararg: a
`
	assertParse(t, expected, src)
}

func TestVarargInvalid(t *testing.T) {
	// not considered a vararg as it is not last, so it is stored
	// as a standard argument.
	src := "fn(*a, b)"
	expected := `
*a ""
b ""
`
	assertParse(t, expected, src)
}

func TestKwarg(t *testing.T) {
	src := "fn(a, **b)"
	expected := `
a ""
Kwarg: b
`
	assertParse(t, expected, src)
}

func TestKwargFirst(t *testing.T) {
	src := "fn(**a)"
	expected := `
Kwarg: a
`
	assertParse(t, expected, src)
}

func TestKwargInvalid(t *testing.T) {
	// not considered a kwarg as it is not last, so it is stored
	// as a standard argument.
	src := "fn(**a, b)"
	expected := `
**a ""
b ""
`
	assertParse(t, expected, src)
}

func TestCombineVarargKwarg(t *testing.T) {
	src := "fn(*a, **b)"
	expected := `
Vararg: a
Kwarg: b
`
	assertParse(t, expected, src)
}

func TestCombineVarargKwargWithOtherArgs(t *testing.T) {
	src := "fn(x=1, y, *a, **b)"
	expected := `
x "1"
y ""
Vararg: a
Kwarg: b
`
	assertParse(t, expected, src)
}

func TestCombineVarargKwargInvalid(t *testing.T) {
	// kwarg must be after vararg, so only vararg is detected here.
	src := "fn(**b, *a)"
	expected := `
**b ""
Vararg: a
`
	assertParse(t, expected, src)
}

func TestCombineVarargKwargInvalid2(t *testing.T) {
	src := "fn(**b, *a, c)"
	expected := `
**b ""
*a ""
c ""
`
	assertParse(t, expected, src)
}

func TestCombineMultiKwarg(t *testing.T) {
	src := "fn(**b, **a)"
	expected := `
**b ""
Kwarg: a
`
	assertParse(t, expected, src)
}

func TestCombineMultiVararg(t *testing.T) {
	src := "fn(*b, *a)"
	expected := `
*b ""
Vararg: a
`
	assertParse(t, expected, src)
}

func TestBlankLines(t *testing.T) {
	src := "\n\n\tfn(a)"
	expected := `
a ""
`
	assertParse(t, expected, src)
}

func TestTooManyLines(t *testing.T) {
	src := "\n\nfn(a)"
	assertParseWithError(t, "", src, errors.TooManyLines, MaxLines(2))
}

func TestOptionalDelimiter1(t *testing.T) {
	src := "fn([a])"
	expected := `
a "..."
`
	assertParse(t, expected, src)
}

func TestOptionalDelimiter2(t *testing.T) {
	src := "fn(a[, b], c)"
	expected := `
a ""
b "..."
c ""
`
	assertParse(t, expected, src)
}

func TestOptionalDelimiter3(t *testing.T) {
	src := "fn(a=1 [, b = 2, c=3], d)"
	expected := `
a "1"
b "2"
c "3"
d ""
`
	assertParse(t, expected, src)
}

func TestOptionalDelimiter4(t *testing.T) {
	src := "fn(a=1 [, b, c], *d)"
	expected := `
a "1"
b "..."
c "..."
Vararg: d
`
	assertParse(t, expected, src)
}

func TestOptionalDelimiter5(t *testing.T) {
	src := "fn(a=1 [, b, c], *d, **e)"
	expected := `
a "1"
b "..."
c "..."
Vararg: d
Kwarg: e
`
	assertParse(t, expected, src)
}

func TestOptionalDelimiterSlash(t *testing.T) {
	// NOTE:
	// - it is kind of unclear what we should do with `optional3` here, is
	//   it required (as indicated by `]`) or optional as indicated by '/'.
	// - we currently encode `requiredKeywordOnly` with a default value which means the
	//   downstream consumers will think that this parameter has a default value and is
	//   thus not required
	// - for an example of all of these being included see https://docs.scipy.org/doc/numpy-1.13.0/reference/generated/numpy.cosh.html#numpy.cosh
	src := "fn(required,/,optional1[, optional2], optional3, *, requiredKeywordOnly)"
	expected := `
required ""
optional1 "..."
optional2 "..."
optional3 "..."
requiredKeywordOnly "..."
	`

	assertParse(t, expected, src)
}

func TestGeneratedParserUpToDate(t *testing.T) {
	testparser.ParserUpToDate(t, "internal/pigeon/parser.peg")
}
