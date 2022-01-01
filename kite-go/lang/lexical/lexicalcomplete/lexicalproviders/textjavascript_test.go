package lexicalproviders

import (
	"fmt"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

func Test_TextJavascript_Basic(t *testing.T) {
	src := "var payload = {b$"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.js")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("body"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextJavascript_Import(t *testing.T) {
	src := "import React from $"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.js")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("'react'"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextJavascript_Quotes(t *testing.T) {
	src := "import React from '$"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.js")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("react"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextJavascript_OpenParen(t *testing.T) {
	src := `response.headers.$
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.js")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(fmt.Sprintf("map(%s)", data.Hole(""))),
		Replace: data.Selection{Begin: 0, End: 0},
	}))

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("map("),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextJavascript_NoSyntaxEndingInParen(t *testing.T) {
	src := `
export const LOAD_DOCS = 'load docs'
export const loadDocs = (language, identifier) => ({
  meta: {
    props: { l$ }
  }
})
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.js")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("language"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("language,"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("language:"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextJavascript_NoExtraSpace(t *testing.T) {
	src := `
function LoginButton(props) {
  return (
    <button o$>
  );
}
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.js")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(fmt.Sprintf("onClick={%s}", data.Hole(""))),
		Replace: data.Selection{Begin: -1, End: 0},
	}))

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(fmt.Sprintf(" onClick={%s}", data.Hole(""))),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextJavascript_InvalidCompletionsAfterSpace(t *testing.T) {
	src1 := `
render() {
  const isLoggedIn = this $
}
`
	src2 := `
render() {
  const isLoggedIn = this$
}
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res1, err := runProvider(t, Text{}, src1, "./src.js")
	require.NoError(t, err)

	require.False(t, res1.containsRoot(data.Completion{
		Snippet: data.NewSnippet(".state"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))

	res2, err := runProvider(t, Text{}, src2, "./src.js")
	require.NoError(t, err)

	require.True(t, res2.containsRoot(data.Completion{
		Snippet: data.NewSnippet("this.state"),
		Replace: data.Selection{Begin: -4, End: 0},
	}))
}

func Test_TextJavaScript_EmptyJSXText(t *testing.T) {
	src := `
render(
  <Provider store={store}>
    <Node $
  </Provider>
)`

	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.js")
	require.NoError(t, err)

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("<Node"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("<Provider"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))

	// SEE: https://kite.quip.com/c0WgAlHPJtoa/Lexical-all-languages#ZYEACAlNJX2
	// require.True(t, res.containsRoot(data.Completion{
	// 	Snippet: data.BuildSnippet(fmt.Sprintf("store={%s}", data.Hole(""))),
	// 	Replace: data.Selection{Begin: 0, End: 0},
	// }))
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(fmt.Sprintf("{...this%s}", data.Hole(""))),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

// TODO: turn back on once https://kite.quip.com/c0WgAlHPJtoa/Lexical-all-languages#ZYEACAlNJX2 is resolved
// func Test_TextJavaScript_NoSpaceInJSXTag(t *testing.T) {
// 	src := `
// <template>
//   <div class="footer-wrapper">
//       <li><a href="something">Animal PublicRoutes<$
// `

// 	initModels(t, lexicalmodels.DefaultModelOptions)
// 	res, err := runProvider(t, Text{}, src, "./src.js")
// 	require.NoError(t, err)

// 	require.True(t, res.containsRoot(data.Completion{
// 		Snippet: data.NewSnippet("/a>"),
// 		Replace: data.Selection{Begin: 0, End: 0},
// 	}))

// 	require.False(t, res.containsRoot(data.Completion{
// 		Snippet: data.NewSnippet("/ a >"),
// 		Replace: data.Selection{Begin: 0, End: 0},
// 	}))
// }

func Test_TextJavaScript_NoUnclosedJsxTag(t *testing.T) {
	src := `
<template>
  <div class=$
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.js")
	require.NoError(t, err)

	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet(`"container">`),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextJavaScript_Collapse(t *testing.T) {
	src := `app.configure('development', funct$)
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.js")
	require.NoError(t, err)

	// NOTE: we do not have the extra newline before the } because we
	// run out of depth
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet(fmt.Sprintf("function () {\n%s}", data.Hole(""))),
		Replace: data.Selection{Begin: -5, End: 0},
	}))
}

func Test_TextJavaScript_StringSubtoken(t *testing.T) {
	src := `
let alpha = 'alpha beta';
let beta = 'alpha $'
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.js")

	// NOTE: this is different from the old JS model (which included the space before beta) because
	// we used to have one giant token for the entire string
	// and then split that into "subtokens" as part of
	// building the "before context" in predict/context.go
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("beta"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextJavaScript_TemplateChars(t *testing.T) {
	src := "<div className={`main $`}>"
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.js")

	// NOTE: this is different from the old JS model (which included the space before $) because
	// we used to have one giant token for the entire string
	// and then split that into "subtokens" as part of
	// building the "before context" in predict/context.go
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet(fmt.Sprintf("${this.props%s}", data.Hole(""))),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextJavaScript_MidToken_Keyword(t *testing.T) {
	src := "for (l$et i = 0;"

	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.js")

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("let"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("let i"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))

	require.Equal(t, "l", res.in.PredictInputs.Prefix)
}

func Test_TextJavaScript_MidToken_KeywordAsIdent(t *testing.T) {
	src := `
function send(letters) {
	l$.forEach(
		letter => {
}
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.js")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("letters"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("letters.forEach"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))

	require.Equal(t, "l", res.in.PredictInputs.Prefix)
}

func Test_TextJavascript_CompletionBetweenQuotes(t *testing.T) {
	src := `
class Home extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      protocol: "$",
    }
  }
}
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.js")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("http"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}
