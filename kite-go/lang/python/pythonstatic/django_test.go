package pythonstatic

import (
	"fmt"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssembler_Django(t *testing.T) {
	src := `
from django.db.models import Model

class Question(Model):
    txt = "some text"


txt1 = Question.objects.all()[0].txt
txt2 = Question.objects.all().filter()[0].txt
id = Question.objects.all()[0].id
pk = Question.objects.all()[0].pk
txt3 = Question._meta.get_field("txt")
`

	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"django.db.models":       keytypes.TypeInfo{Kind: keytypes.ModuleKind},
		"django.db.models.Model": keytypes.TypeInfo{Kind: keytypes.TypeKind},
	})

	base, err := manager.PathSymbol(pythonimports.NewPath("django", "db", "models", "Model"))
	require.NoError(t, err)

	modelAddr := pythontype.Address{
		File: "src.py",
		Path: pythonimports.NewDottedPath("Question"),
	}

	model := &pythontype.SourceClass{
		Bases:   []pythontype.Value{pythontype.NewExternal(base, manager)},
		Members: pythontype.NewSymbolTable(modelAddr, nil),
	}

	s := model.Members.Create("objects")
	s.Value = pythontype.NewManager(model)

	s = model.Members.Create("id")
	s.Value = pythontype.IntInstance{}

	s = model.Members.Create("pk")
	s.Value = pythontype.IntInstance{}

	s = model.Members.Create("txt")
	s.Value = pythontype.StrConstant("some text")

	assertTypes(t, src, manager, map[string]pythontype.Value{
		"txt1": pythontype.StrConstant("some text"),
		"txt2": pythontype.StrConstant("some text"),
		"txt3": pythontype.Builtins.Str,
		"id":   pythontype.IntInstance{},
		"pk":   pythontype.IntInstance{},
	})
}

func TestAssembler_DjangoView(t *testing.T) {
	views := `
from django.db.models import Model

def view(req):
	return req

class View():
	def view(self, r):
		return r

r = view(None)
rr = View().view(None)
`
	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"django.http.request.HttpRequest": keytypes.TypeInfo{Kind: keytypes.TypeKind},
		"django.db.models.Model":          keytypes.TypeInfo{Kind: keytypes.TypeKind},
	})

	req, err := manager.PathSymbol(djangoRequestAddress.Path)
	require.NoError(t, err)

	src := map[string]string{"/code/views.py": views}

	val := pythontype.UniteNoCtx(pythontype.ExternalInstance{TypeExternal: pythontype.NewExternal(req, manager)}, pythontype.Builtins.None)
	assertBatchTypes(t, src, manager, map[string]pythontype.Value{
		"r":  val,
		"rr": val,
	})
}

func TestAssembler_DjangoAdminActionRequest(t *testing.T) {
	admin := `
from django.db.models import Model

def action(model=None,req, queryset):
	return req

class admin():
	def action(self, req, queryset):
		return req

r = action(None,None,None)
rr = admin().action(None,None)
	`

	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"django.http.request.HttpRequest": keytypes.TypeInfo{Kind: keytypes.TypeKind},
		"django.db.models.Model":          keytypes.TypeInfo{Kind: keytypes.TypeKind},
	})

	req, err := manager.PathSymbol(djangoRequestAddress.Path)
	require.NoError(t, err)

	src := map[string]string{"/code/admin.py": admin}

	val := pythontype.UniteNoCtx(pythontype.ExternalInstance{TypeExternal: pythontype.NewExternal(req, manager)}, pythontype.Builtins.None)
	assertBatchTypes(t, src, manager, map[string]pythontype.Value{
		"r":  val,
		"rr": val,
	})
}

func TestAssembler_DjangoAdminActionQuerySet(t *testing.T) {
	admin := `
from django.db.models import Model

class Question(Model):
    txt = "some text"

def action(question,req,queryset):
	return queryset
	
t = action(Question,None,None)[0].txt
	`

	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"django.db.models.Model": keytypes.TypeInfo{Kind: keytypes.TypeKind},
	})

	src := map[string]string{"/code/admin.py": admin}

	assertBatchTypes(t, src, manager, map[string]pythontype.Value{
		"t": pythontype.StrConstant("some text"),
	})
}

func TestAssembler_DjangoShortcuts(t *testing.T) {
	src := `
from django.db.models import Model
from django.shortcuts import get_object_or_404, get_list_or_404

class M(Model): pass

a = get_object_or_404(M)
b = get_list_or_404(M)
`
	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"django.db.models.Model":             keytypes.TypeInfo{Kind: keytypes.TypeKind},
		"django.shortcuts.get_object_or_404": keytypes.TypeInfo{Kind: keytypes.FunctionKind},
		"django.shortcuts.get_list_or_404":   keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	})
	syms := assertTypes(t, src, manager, nil)

	a := syms["a"]
	b := syms["b"]
	require.NotNil(t, a)
	require.NotNil(t, b)

	assert.Equal(t, "src-instance:src.py:M", fmt.Sprintf("%v", a.Value))
	assert.Equal(t, "[src-instance:src.py:M]", fmt.Sprintf("%v", b.Value))
}

func TestAssembler_NilModelOptions(t *testing.T) {
	src := `
from django.db.models.options import Options

opts = Options()
f = opts.get_field("foo")
`

	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"django.db.models.options.Options": keytypes.TypeInfo{Kind: keytypes.TypeKind},
	})

	assertTypes(t, src, manager, map[string]pythontype.Value{
		"opts": pythontype.NewOptions(nil),
		"f":    pythontype.Builtins.None,
	})
}

func TestAssembler_DjangoGetModel(t *testing.T) {
	models := `
from django.db.models import Model
class Book(Model):
	Name = "Book"
	`

	init := ``

	src := `
from django.db.models import get_model

name = get_model("blog", "Book").Name
`

	srcs := map[string]string{
		"/code/blog/__init__.py": init,
		"/code/blog/models.py":   models,
		"/code/src.py":           src,
	}

	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"django.db.models":           keytypes.TypeInfo{Kind: keytypes.ModuleKind},
		"django.db.models.Model":     keytypes.TypeInfo{Kind: keytypes.TypeKind},
		"django.db.models.get_model": keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	})

	assertBatchTypes(t, srcs, manager, map[string]pythontype.Value{
		"name": pythontype.StrConstant("Book"),
	})
}

func TestAssembler_DjangoGetNextGetPrev(t *testing.T) {
	src := `
from django.db import models
class Book(models.Model):
	pubdate = models.fields.DateField()
	releasedate = models.fields.DateTimeField()
	title = "hello"

a = Book.get_next_by_pubdate().title
b = Book.get_previous_by_pubdate().title
c = Book.get_next_by_releasedate().title
d = Book.get_previous_by_releasedate().title
`

	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"django.db.models.Model":                keytypes.TypeInfo{Kind: keytypes.TypeKind},
		"django.db.models.fields.DateField":     keytypes.TypeInfo{Kind: keytypes.TypeKind},
		"django.db.models.fields.DateTimeField": keytypes.TypeInfo{Kind: keytypes.TypeKind},
	})

	hello := pythontype.StrConstant("hello")

	assertTypes(t, src, manager, map[string]pythontype.Value{
		"a": hello,
		"b": hello,
		"c": hello,
		"d": hello,
	})
}
