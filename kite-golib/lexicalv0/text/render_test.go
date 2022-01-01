package text

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type renderTC struct {
	Desc     string
	Before   string
	Src      string
	PH       string
	Expected string
}

func assertRender(t *testing.T, l lang.Language, tcs []renderTC) {
	for i, tc := range tcs {
		toks, err := NewLexer().Lex([]byte(tc.Src))
		require.NoError(t, err)
		before, err := NewLexer().Lex([]byte(tc.Before))
		require.NoError(t, err)

		expected := strings.ReplaceAll(tc.Expected, tc.PH, "\x02\x03")

		actual, _ := Render(l, before, toks)
		assert.Equal(t, expected, actual.ForFormat(), "test case %d: %s", i, tc.Desc)
	}
}

func TestRender_Golang(t *testing.T) {
	tcs := []renderTC{
		{
			Desc:     "no auto close if completion does not end in one of: alpha numeric, underscore, whitespace, or open/close syntax",
			Src:      "foo(bar,",
			PH:       "$",
			Expected: "foo(bar,",
		},
		{
			Desc:     "auto close if completion ends in hspace",
			Src:      "foo(bar ",
			PH:       "$",
			Expected: "foo(bar $)",
		},
		{
			Desc:     "auto close if completion ends in vspace",
			Src:      "func main() {\n",
			PH:       "$",
			Expected: "func main() {\n$}",
		},
		{
			Desc:     "auto close if completion ends in alpha numeric",
			Src:      "foo(bar",
			PH:       "$",
			Expected: "foo(bar$)",
		},
		{
			Desc:     "no auto close if completion ends in dash",
			Src:      "foo(bar-",
			PH:       "$",
			Expected: "foo(bar-",
		},
		{
			Desc:     "auto close if completion ends in underscore",
			Src:      "foo(bar_",
			PH:       "$",
			Expected: "foo(bar_$)",
		},
		{
			Desc:     "auto close if completion ends in close syntax",
			Src:      "foo((bar)",
			PH:       "$",
			Expected: "foo((bar)$)",
		},
		{
			Desc:     "auto close if completion ends in open syntax",
			Src:      "foo(",
			PH:       "$",
			Expected: "foo($)",
		},
		{
			Desc:     "auto close mixed",
			Src:      "foo({[",
			PH:       "$",
			Expected: "foo({[$]$}$)",
		},
		{
			Desc:     "auto close multiple",
			Src:      "foo(((((",
			PH:       "$",
			Expected: "foo((((($)$)$)$)$)",
		},
		{
			Desc:     "no auto close but contains parens",
			Src:      "foo()",
			PH:       "$",
			Expected: "foo()",
		},
		{
			Desc:     "abandon if mismatch",
			Src:      "foo(}",
			PH:       "$",
			Expected: "",
		},
		{
			Desc:     "abandon is no paired quotes",
			Src:      "'foo",
			PH:       "$",
			Expected: "",
		},
		{
			Desc:     "abandon if single closing",
			Src:      "foo)",
			PH:       "$",
			Expected: "",
		},
		{
			Desc:     "no single close paren",
			Src:      "foo[])",
			PH:       "$",
			Expected: "",
		},
		{
			Desc:     "no single close paren",
			Src:      "foo[])",
			PH:       "$",
			Expected: "",
		},
		{
			Desc:     "no single quotes",
			Src:      "foo'bar",
			PH:       "$",
			Expected: "",
		},
	}
	assertRender(t, lang.Golang, tcs)
}

func TestRender_Java(t *testing.T) {
	tcs := []renderTC{
		{
			Desc:     "do not enforce <>",
			Before:   "if ",
			Src:      "(a < b)",
			PH:       "$",
			Expected: "(a < b)",
		},
		{
			Desc:     "enforce <>: no single closing one",
			Before:   "class <",
			Src:      "T>",
			PH:       "$",
			Expected: "",
		},
		{
			Desc:     "enforce <>: maybe auto close",
			Before:   "class ",
			Src:      "<T",
			PH:       "$",
			Expected: "<T$>",
		},
		{
			Desc:     "enforce <>: maybe auto close",
			Before:   "public static <T ",
			Src:      "extends Comparable<T",
			PH:       "$",
			Expected: "extends Comparable<T$>",
		},
	}
	assertRender(t, lang.Java, tcs)
}

func TestRender_HTML(t *testing.T) {
	tcs := []renderTC{
		{
			Desc:     "enforce <>: allowed",
			Before:   "<h> something ",
			Src:      "</h>",
			PH:       "$",
			Expected: "</h>",
		},
		{
			Desc:     "enforce <>: not allow",
			Before:   "<h> something <",
			Src:      "/h>",
			PH:       "$",
			Expected: "",
		},
		{
			Desc:     "enforce <>: maybe auto close",
			Before:   "<h> something ",
			Src:      "</h",
			PH:       "$",
			Expected: "</h$>",
		},
	}
	assertRender(t, lang.HTML, tcs)
}
