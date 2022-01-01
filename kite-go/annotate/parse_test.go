package annotate

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsIdentifier(t *testing.T) {
	assert.True(t, isIdentifier("abc"))
	assert.True(t, isIdentifier("abc_xyz123"))
	assert.False(t, isIdentifier("abc.def"))
	assert.False(t, isIdentifier(""))
	assert.False(t, isIdentifier("1abc"))
	assert.False(t, isIdentifier("foo(1, 2)"))
	assert.False(t, isIdentifier("bar['x']"))
}

func checkParse(t *testing.T, src string, language lang.Language, expectedPresentation, expectedRunnable string) {
	parsed, err := ParseExample(src, language)
	require.NoError(t, err)
	if parsed.Runnable == parsed.Presentation {
		t.Logf("presentation == runnable == \n%s\n", parsed.Presentation)
	}
	assert.Equal(t, expectedPresentation, parsed.Presentation, "parsed presentation is wrong")
	assert.Equal(t, expectedRunnable, parsed.Runnable, "parsed runnable is wrong")
	assert.NotEqual(t, parsed.Runnable, parsed.Presentation, "parsed runnable and presentation should differ")
	assert.Equal(t, src, parsed.Original)
}

func TestParseSimplePythonExample(t *testing.T) {
	src := `
a = 10
a += 5
##kite.show_plaintext_str(expression="a", value=a)`

	expectedPresentation := `
a = 10
a += 5`

	expectedRunnable := `from kite import kite

a = 10
a += 5
kite.show_plaintext_str(expression="a", value=a)`

	checkParse(t, src, lang.Python, expectedPresentation, expectedRunnable)
}

func TestParseSimpleBashExample(t *testing.T) {
	src := `
date
uname`

	expectedPresentation := `
date
uname`

	expectedRunnable := string(bashPresentationAPI) + `
kite_line 15

kite_line 17
date
kite_line 19
uname`

	checkParse(t, src, lang.Bash, expectedPresentation, expectedRunnable)
}

func TestParseBashExampleWithBlock(t *testing.T) {
	src := `
for CITY in SanFrancisco Berkeley PaloAlto
do
    echo $CITY
done`

	expectedPresentation := src

	expectedRunnable := string(bashPresentationAPI) + `
kite_line 15

kite_line 20
for CITY in SanFrancisco Berkeley PaloAlto
do
    echo $CITY
done`

	checkParse(t, src, lang.Bash, expectedPresentation, expectedRunnable)
}

func TestParseBashExampleWithNestedBlock(t *testing.T) {
	src := `
for CITY in SanFrancisco Berkeley Dublin
do
    for PERSON in Noah Tarak Hrysoula
    do
        echo $PERSON in $CITY
    done
done`

	expectedPresentation := src

	expectedRunnable := string(bashPresentationAPI) + `
kite_line 15

kite_line 23
for CITY in SanFrancisco Berkeley Dublin
do
    for PERSON in Noah Tarak Hrysoula
    do
        echo $PERSON in $CITY
    done
done`

	checkParse(t, src, lang.Bash, expectedPresentation, expectedRunnable)
}

func TestParseBashExampleDoubleBackslashLineEnding(t *testing.T) {
	src := `
echo C:\\
uname`

	expectedPresentation := `
echo C:\\
uname`

	expectedRunnable := string(bashPresentationAPI) + `
kite_line 15

kite_line 17
echo C:\\
kite_line 19
uname`

	checkParse(t, src, lang.Bash, expectedPresentation, expectedRunnable)
}

func TestParseIndented(t *testing.T) {
	src := `
if foo:
	## x = 1
	y = 2`

	expectedPresentation := `
if foo:
	y = 2`

	expectedRunnable := `from kite import kite

if foo:
	x = 1
	y = 2`

	checkParse(t, src, lang.Python, expectedPresentation, expectedRunnable)
}

func TestParseBashExampleWithWhitespaceAfterBackslash(t *testing.T) {
	// note the whitespace after the '\'' continuation character:
	src := strings.TrimSpace(`
echo hello \  
date`)

	expectedPresentation := strings.TrimSpace(`
echo hello \  
date`)

	expectedRunnable := string(bashPresentationAPI) + `
kite_line 15
echo hello \  
kite_line 17
date`

	checkParse(t, src, lang.Bash, expectedPresentation, expectedRunnable)
}

func TestTrimLeader(t *testing.T) {
	assert.Equal(t, "foo", trimLeader("## foo", "##"))
	assert.Equal(t, "  foo", trimLeader("  ## foo", "##"))
	assert.Equal(t, "", trimLeader(" foo", "##"))
}
