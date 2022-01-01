package clientapp

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/stretchr/testify/assert"
)

func Test_EnabledLanguages(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-lang")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)
	mgr := settings.NewTestManager()

	langs := enabledLanguages(mgr)

	for _, l := range lexicalv0.WebGroup.Langs {
		assert.True(t, hasLang(l, langs))
	}

	for _, l := range lexicalv0.JavaPlusPlusGroup.Langs {
		assert.True(t, hasLang(l, langs))
	}

	for _, l := range lexicalv0.CStyleGroup.Langs {
		assert.True(t, hasLang(l, langs))
	}
	//disabled by default
	assert.False(t, hasLang(lang.Perl, langs))
}

func hasLang(requested lang.Language, langs []lang.Language) bool {
	for _, l := range langs {
		if requested == l {
			return true
		}
	}

	return false
}
