package lexicalproviders

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPython_Basic(t *testing.T) {
	src := "import nu$"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("numpy as np"),
		Replace: data.Selection{Begin: -2, End: 0},
	}))
}

func TestPython_Placeholders(t *testing.T) {
	src := `
	import json

	f = op$
	`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet(fmt.Sprintf("open(%s)", data.Hole(""))),
		Replace: data.Selection{Begin: -2, End: 0},
	}))
}

func TestPython_NoCompletionInComment(t *testing.T) {
	src := "a = 5 # Some comments$"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.Empty(t, res.out)
}

func TestPython_NoCompletionInString(t *testing.T) {
	t.SkipNow()
	src := `message = "hello wo$"`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.Empty(t, res.out)
}

func TestPython_NoCompletionInIncompleteString(t *testing.T) {
	t.SkipNow()
	src := `message = "hello wo$`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.Empty(t, res.out)
}

func TestPython_OpenParen(t *testing.T) {
	src := `
	import requests

	requests.get$
	`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	expected := append([]string{},
		"get(",
		data.HoleWithPlaceholderMarks(""),
		")",
	)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(strings.Join(expected, "")),
		Replace: data.Selection{Begin: -3, End: 0},
	}))
}

func TestPython_NoPlaceholderMatchingPrefix(t *testing.T) {
	src := `
def fn():
    s$
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	disallowed := append([]string{},
		data.HoleWithPlaceholderMarks("str"),
		"\n",
		"    return",
	)

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(strings.Join(disallowed, "")),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func TestPython_Suppress_1(t *testing.T) {
	src := `
import requests

requests.get()$
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.Empty(t, res.out)
}

func TestPython_Lambda(t *testing.T) {
	src := `
website_sources.sort(k$)
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("key=lambda x"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func TestPython_CallNewline(t *testing.T) {
	src := `
import pandas as pd

my_df = pd.DataFrame()
my_df.pivot_table(da$, aggfunc=str, columns=str, values=str)
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet(fmt.Sprintf(`data=%s,
    index`, data.HolePH("str"))),
		Replace: data.Selection{Begin: -2, End: 0},
	}))
}

func TestPython_Suppress_2(t *testing.T) {
	src := `
with open("text.txt") as f:$
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.Empty(t, res.out)
}

func TestPython_Suppress_3(t *testing.T) {
	src := `
import json
f = open($)
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.NotEmpty(t, res.out)
}

func TestPython_Suppress_4(t *testing.T) {
	src := `
import requests

requests.get()
$
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.Empty(t, res.out)
}

func TestPython_NewLineFilter(t *testing.T) {
	srcs := []string{
		`
class Data:
    def __init__(self, queries, options):
        self.queries = queries$
`,
		`
import alpha


def beta(gamma):
	delta = epsilon(gamma)
	return d$


def epsilon(gamma):
	return alpha.phi(gamma)
`,
	}
	for _, src := range srcs {
		initModels(t, lexicalmodels.DefaultModelOptions)
		res := requireRes(t, Text{}, src, "./src.py")

		for _, mcs := range res.out {
			for _, mc := range mcs {
				assert.False(t, strings.HasPrefix(mc.Completion.Snippet.Text, "\n"))
				assert.False(t, strings.HasPrefix(mc.Completion.Snippet.Text, "queries\n"))
				assert.False(t, strings.Contains(mc.Completion.Snippet.Text, "\n\n"))
			}
		}
	}
}

func TestPython_EmptyLine(t *testing.T) {
	src := `def alpha():
    $`
	initModels(t, lexicalmodels.DefaultModelOptions)
	requireRes(t, Text{}, src, "./src.py")
}

func TestPython_FirstLineOfBlock(t *testing.T) {
	src := `
	import scrapy

	class BookSpider(scrapy.Spider):
		n$
	`

	initModels(t, lexicalmodels.DefaultModelOptions)

	expected := "name = scrapy.Field"

	res := requireRes(t, Text{}, src, "./src.py")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(expected),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func TestPython_MultiLine_Basic(t *testing.T) {
	src := `
	import scrapy
	class BookSpider(scrapy.Spider):
		name = 'name'
		i$
	`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("id = None"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func TestPython_MultiLine_Nested(t *testing.T) {
	src := `
	class BookSpider(scrapy.Spider):
		def __init__(self, url):
			self.url$
	`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("url = url"),
		Replace: data.Selection{Begin: -3, End: 0},
	}))
}

func TestPython_Ending_With_Colon(t *testing.T) {
	src := `
def ma$
`

	initModels(t, lexicalmodels.DefaultModelOptions)
	expected := "main():"
	res := requireRes(t, Text{}, src, "./src.py")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(expected),
		Replace: data.Selection{Begin: -2, End: 0},
	}))
}

func TestPython_None(t *testing.T) {
	src := `
def alpha(beta):
    if beta is N$
`
	expected := "None"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(expected),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func TestPython_True(t *testing.T) {
	src := `
def alpha(beta):
    if beta is None:
        return T$
`
	expected := "True"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(expected),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func TestPython_False(t *testing.T) {
	src := `
def alpha(beta):
    if beta is None:
        return F$
`
	expected := "False"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(expected),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func TestPython_Stable_Random_Seed(t *testing.T) {
	src1 := `import os
import sys
i$
`
	src2 := `import os
import sys
im$
`

	initModels(t, lexicalmodels.DefaultModelOptions)

	res1 := requireRes(t, Text{}, src1, "./src.py")
	res2 := requireRes(t, Text{}, src2, "./src.py")

	hit1 := res1.containsRoot(data.Completion{
		Snippet: data.NewSnippet("import subprocess"),
		Replace: data.Selection{Begin: -1, End: 0},
	})
	hit2 := res2.containsRoot(data.Completion{
		Snippet: data.NewSnippet("import subprocess"),
		Replace: data.Selection{Begin: -2, End: 0},
	})

	require.True(t, hit1 == hit2)
}

func TestPython_ElseNoSpace(t *testing.T) {
	src := `
def alpha(beta):
    if beta:
        gamma()
    e$
`
	expected := "else:"
	unexpected := "else :"

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(expected),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(unexpected),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func TestPython_ElseSpace(t *testing.T) {
	src := `
def alpha(beta, gamma):
    delta = beta if beta > 0 else None
    epsilon = gamma if gamma > 0 e$
`
	expected := "else None"
	unexpected := "elseNone"

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(expected),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(unexpected),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func TestPython_TryNoSpace(t *testing.T) {
	src := `
def fetch(obj):
    t$
`
	expected := "try:"
	unexpected := "try :"

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(expected),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(unexpected),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func TestPython_FinallyNoSpace(t *testing.T) {
	src := `
def alpha(beta):
    try:
        gamma()
    f$
`
	expected := "finally:"
	unexpected := "finally :"

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(expected),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(unexpected),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func TestPython_MidToken_Ident(t *testing.T) {
	src := "def m$e():"

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("main"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("main()"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))

	require.Equal(t, "m", res.in.PredictInputs.Prefix)
}

func TestPython_MidToken_Keyword(t *testing.T) {
	src := "d$ef main():"

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("def"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("def main"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))

	require.Equal(t, "d", res.in.PredictInputs.Prefix)
}

func TestPython_NoTrailingSpace(t *testing.T) {
	src := `
total = 0
f$
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("for "),
		Replace: data.Selection{Begin: -1, End: 0},
	}))

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("for"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func TestPython_MidToken_KeywordAsIdent(t *testing.T) {
	src := `
def run(fork):
    if fo$r is None:
        return
`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("fork"),
		Replace: data.Selection{Begin: -2, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("fork is"),
		Replace: data.Selection{Begin: -2, End: 0},
	}))

	require.Equal(t, "fo", res.in.PredictInputs.Prefix)
}

func TestPython_MidFirstToken(t *testing.T) {
	src := "im$po"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.py")
	require.True(t, len(res.in.PredictInputs.Prefix) <= res.in.SelectedBuffer.Selection.Begin)
	require.True(t, res.in.SelectedBuffer.Selection.Begin <= res.in.SelectedBuffer.Selection.End)
	require.True(t, res.in.SelectedBuffer.Selection.End <= len(res.in.SelectedBuffer.Text()))
}
