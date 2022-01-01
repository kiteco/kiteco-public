package lexicalproviders

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

func TestVue_JSTag(t *testing.T) {
	src := `
<template>
    <section class="recipes">
        <Pagination :list-data="recipes"/>
     </section>
</template>

<script>
import React from $
</script>
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.vue")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("'react'"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func TestVue_Incomplete_JSTag(t *testing.T) {
	src := `
<template>
    <section class="recipes">
        <Pagination :list-data="recipes"/>
     </section>
</template>

<script>
import React from $
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.vue")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("'react'"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func TestVue_Template(t *testing.T) {
	src := `
<template>
  <div>
    <section $>
  </div>
</template>
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.vue")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("class"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func TestVue_Incomplete_Template(t *testing.T) {
	src := `
<template>
  <div>
    <section $>
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res, err := runProvider(t, Text{}, src, "./src.vue")
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("class"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}
