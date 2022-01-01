package lexicalv0

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLangGroupFromName(t *testing.T) {
	type tc struct {
		Desc      string
		Name      string
		Lexer     lang.Language
		Langs     []lang.Language
		ShouldErr bool
	}

	tcs := []tc{
		{
			Desc:  "basic parsing web group",
			Name:  "text__javascript-jsx-vue-css-html-less-typescript-tsx",
			Lexer: lang.Text,
			Langs: WebGroup.Langs,
		},
		{
			Desc:  "basic parsing java group",
			Name:  "text__java-scala-kotlin",
			Lexer: lang.Text,
			Langs: JavaPlusPlusGroup.Langs,
		},
		{
			Desc:  "basic parsing c style group",
			Name:  "text__c-cpp-objectivec-csharp",
			Lexer: lang.Text,
			Langs: CStyleGroup.Langs,
		},
		{
			Desc:      "duplicate elems",
			Name:      "text__javascript-javascript",
			ShouldErr: true,
		},
		{
			Desc:      "test no base",
			Name:      "__java",
			ShouldErr: true,
		},
		{
			Desc:  "check just base old lang",
			Name:  "go",
			Lexer: lang.Golang,
			Langs: []lang.Language{lang.Golang},
		},
		{
			Desc:  "check all langs",
			Name:  "text__python-go-javascript-jsx-vue-css-html-less-typescript-tsx-java-scala-kotlin-c-cpp-objectivec-csharp-php-ruby-bash",
			Lexer: lang.Text,
			Langs: AllLangsGroup.Langs,
		},
		{
			Desc:  "check misc langs",
			Name:  "text__python-go-php-ruby-bash",
			Lexer: lang.Text,
			Langs: MiscLangsGroup.Langs,
		},
	}

	for i, tc := range tcs {
		actual, err := LangGroupFromName(tc.Name)
		if tc.ShouldErr {
			assert.Error(t, err, "test case %d: %s", i, tc.Desc)
		} else {
			expected := LangGroup{Lexer: tc.Lexer, Langs: tc.Langs}
			assert.Equal(t, expected, actual, "test case %d: %s", i, tc.Desc)
		}
	}
}

func TestLangGroupFromNameAndNameCompatible(t *testing.T) {
	var nativeLangs []lang.Language
	for l := range lang.LanguageTags {
		nativeLangs = append(nativeLangs, l)
	}

	for i, l := range nativeLangs {
		others := append([]lang.Language{}, nativeLangs[:i]...)
		others = append(others, nativeLangs[i+1:]...)
		g := NewLangGroup(l, others...)

		gg, err := LangGroupFromName(g.Name())
		require.NoError(t, err)

		assert.Equal(t, g, gg)
	}

}

func TestBackwardsCompatWithLangPackage(t *testing.T) {
	for l := range lang.LanguageTags {
		// old to new
		g, err := LangGroupFromName(l.Name())
		require.NoError(t, err)

		assert.Equal(t, l, g.Lexer)
		require.Len(t, g.Langs, 1)
		assert.Equal(t, l, g.Langs[0])

		// make sure new to old works
		g = NewLangGroup(l, l)
		assert.Equal(t, l.Name(), g.Name())

		ll := lang.FromName(g.Name())
		assert.Equal(t, l, ll)
	}
}
