package traindata

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/stretchr/testify/assert"
)

func assertSplitNameLiteral(t *testing.T, lit string, expected ...string) {
	actual := SplitNameLiteral(lit)
	t.Logf("literal: %s\n", lit)
	assert.Equal(t, expected, actual)
}

func TestSplitNameLiteralNoSplit(t *testing.T) {
	assertSplitNameLiteral(t, "foo", "foo")
	assertSplitNameLiteral(t, "Foo", "foo")
	assertSplitNameLiteral(t, "FOO", "foo")
	assertSplitNameLiteral(t, "f", "f")
}

func TestSplitNameLiteralAllUnderscore(t *testing.T) {
	assertSplitNameLiteral(t, "_", "_")
	assertSplitNameLiteral(t, "__", "__")
	assertSplitNameLiteral(t, "___", "___")
}
func TestSplitNameLiteralSnakeCase(t *testing.T) {
	assertSplitNameLiteral(t, "foo_bar_car", "foo", "bar", "car")
	assertSplitNameLiteral(t, "_foo_bar_", "foo", "bar")
	assertSplitNameLiteral(t, "foo__bar_car", "foo", "bar", "car")
	assertSplitNameLiteral(t, "foo_bar_c", "foo", "bar", "c")
	assertSplitNameLiteral(t, "__foo__", "foo")
}

func TestSplitNameLiteralCamelCase(t *testing.T) {
	assertSplitNameLiteral(t, "fooBarCar", "foo", "bar", "car")
	assertSplitNameLiteral(t, "FooBarCar", "foo", "bar", "car")
}

func TestSplitNameLiteralNumbers(t *testing.T) {
	assertSplitNameLiteral(t, "foo123", "foo", "123")
	assertSplitNameLiteral(t, "foo123bar", "foo", "123", "bar")
}

func TestSplitNameLiteralMixSnakeAndCamel(t *testing.T) {
	assertSplitNameLiteral(t, "foo_BarCar_Star", "foo", "bar", "car", "star")
}

func TestSplitNameNonASCIIChararacter(t *testing.T) {
	assertSplitNameLiteral(t, "iü_iü", "iü", "iü")
	assertSplitNameLiteral(t, "IüÜi", "iü", "üi")
}

func TestSplitNameNonLatinChararacter(t *testing.T) {
	assertSplitNameLiteral(t, "A学生Bc", "a", "学生", "bc")
}

func TestNodeLabel(t *testing.T) {
	assert.Equal(t, "KITE_IMPORTNAMESTMT", ASTNodeType(&pythonast.ImportNameStmt{}))
}
