package pythondocs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	moduleName = "PyQt4"
)

func TestParsePyQt4_Class(t *testing.T) {
	filepath := "./test/qtextobject.html"
	r, err := os.Open(filepath)
	if err != nil {
		t.Error(err)
		return
	}
	defer r.Close()
	module := ParsePyQt4HTML(r, filepath, false)
	if module == nil {
		t.Error("Parser returned no results")
		return
	}

	require.Equal(t, module.Name, "PyQt4", "module name should be PyQt4")

	ident := moduleName + ".QtGui.QTextObject"

	expected := []LangEntity{
		{Kind: ClassKind, Module: moduleName, Ident: moduleName + ".QtGui", Sel: "QTextObject"},
		{Kind: MethodKind, Module: moduleName, Ident: ident, Sel: "__init__"},
		{Kind: MethodKind, Module: moduleName, Ident: ident, Sel: "document"},
		{Kind: MethodKind, Module: moduleName, Ident: ident, Sel: "format"},
		{Kind: MethodKind, Module: moduleName, Ident: ident, Sel: "formatIndex"},
		{Kind: MethodKind, Module: moduleName, Ident: ident, Sel: "objectIndex"},
		{Kind: MethodKind, Module: moduleName, Ident: ident, Sel: "setFormat"},
	}

	for _, e := range expected {
		ok, err := searchModule(module, &e)
		require.NoError(t, err)
		require.True(t, ok, "Could not find expected %+v in returned module\n", e)
	}
}

func TestParsePyQt4_Module(t *testing.T) {
	filepath := "./test/qtscript.html"
	r, err := os.Open(filepath)
	if err != nil {
		t.Error(err)
		return
	}
	defer r.Close()
	module := ParsePyQt4HTML(r, filepath, false)
	if module == nil {
		t.Error("Parser returned no results")
		return
	}

	require.Equal(t, module.Name, moduleName, "module name should be PyQt4")

	ident := moduleName + ".QtScript"

	expected := []LangEntity{
		{Kind: FunctionKind, Module: moduleName, Ident: ident, Sel: "qScriptConnect"},
		{Kind: FunctionKind, Module: moduleName, Ident: ident, Sel: "qScriptDisconnect"},
	}

	for _, e := range expected {
		ok, err := searchModule(module, &e)
		require.NoError(t, err)
		require.True(t, ok, "Could not find expected %+v in returned module\n", e)
	}
}
