package stats

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/stretchr/testify/require"
)

func mockManager(t *testing.T, paths map[string]keytypes.Kind) pythonresource.Manager {
	infos := make(map[string]keytypes.TypeInfo)
	for path, kind := range paths {
		infos[path] = keytypes.TypeInfo{Kind: kind}
	}

	return pythonresource.MockManager(t, infos)
}

func assertExtract(t *testing.T, src string, rm pythonresource.Manager, client *typeinduction.Client, expected PathCounts) {
	params := Params{
		Client:  client,
		Manager: rm,
	}

	actual, err := Extract(params, []byte(src))
	require.NoError(t, err)

	for path, counts := range expected {
		if _, found := actual[path]; !found {
			t.Errorf("expected path `%s` not found", path)
		} else if *counts != *actual[path] {
			t.Errorf("for `%s` expected %+v but got %+v\n", path, *counts, *actual[path])
		}
	}

	for name := range actual {
		if _, found := expected[name]; !found {
			t.Errorf("got extra stats for `%s`\n", name)
		}
	}
}

func TestExtract(t *testing.T) {
	src := `
import foo
import bar.car
from bar.car import mar, star

a = foo
b = foo.bar
c = foo.bar
e = star
f = mar
`

	expected := PathCounts{
		"foo":          {Import: 1, Name: 1},
		"foo.bar":      {Attribute: 2},
		"bar.car":      {Import: 1},
		"bar.car.mar":  {Import: 1, Name: 1},
		"bar.car.star": {Import: 1, Name: 1},
	}

	rm := pythonresource.MockManager(t, nil, "foo.bar", "bar.car.star", "bar.car.mar")

	assertExtract(t, src, rm, nil, expected)
}

func TestCalls(t *testing.T) {
	src := `
import re
x = re.compile()
y = x.match()
`

	rm := pythonresource.MockManager(t, pythonresource.InfosFromKinds(map[string]pythonimports.Kind{
		"re.compile":    pythonimports.Function,
		"pattern":       pythonimports.Type,
		"pattern.match": pythonimports.Function,
		"matchobj":      pythonimports.Type,
	}))

	client := typeinduction.MockClient(rm, map[string]string{
		"re.compile":    "pattern",
		"pattern.match": "matchobj",
	})

	assertExtract(t, src, rm, client, PathCounts{
		"re":            {Import: 1},
		"re.compile":    {Attribute: 1},
		"pattern":       {Expr: 1},
		"pattern.match": {Attribute: 1},
		"matchobj":      {Expr: 1},
	})
}

func TestAttrCalls(t *testing.T) {
	src := `
import foo
foo()
foo().car
foo.mar()

import re, batch
x = re.compile()
batch(x.match)
`

	rm := pythonresource.MockManager(t, pythonresource.InfosFromKinds(map[string]pythonimports.Kind{
		"batch":         pythonimports.Function,
		"foo":           pythonimports.Function,
		"foo.bar":       pythonimports.Type,
		"foo.bar.car":   pythonimports.Type,
		"foo.star":      pythonimports.Type,
		"foo.mar":       pythonimports.Function,
		"re.compile":    pythonimports.Function,
		"pattern":       pythonimports.Type,
		"pattern.match": pythonimports.Function,
		"matchobj":      pythonimports.Type,
	}))

	client := typeinduction.MockClient(rm, map[string]string{
		"foo":           "foo.bar",
		"foo.mar":       "foo.star",
		"re.compile":    "pattern",
		"pattern.match": "matchobj",
	})

	assertExtract(t, src, rm, client, PathCounts{
		"foo":           {Import: 1, Name: 1}, // foo, foo()
		"foo.bar":       {Expr: 1},            // foo()
		"foo.bar.car":   {Attribute: 1},       // foo().car
		"foo.mar":       {Attribute: 1},       // foo.mar
		"foo.star":      {Expr: 1},            // foo.mar()
		"re":            {Import: 1},
		"re.compile":    {Attribute: 1},
		"pattern":       {Expr: 1},
		"pattern.match": {Attribute: 1},
		"batch":         {Import: 1, Name: 1},
	})
}

func TestNestedAttrs(t *testing.T) {
	src := `
import re, foo
def test():
    x = re.compile()
    foo(x.match.a, x.search.b())
`

	rm := pythonresource.MockManager(t, pythonresource.InfosFromKinds(map[string]pythonimports.Kind{
		"foo":            pythonimports.Function,
		"re.compile":     pythonimports.Function,
		"pattern":        pythonimports.Type,
		"pattern.match":  pythonimports.Function,
		"pattern.search": pythonimports.Function,
	}))

	client := typeinduction.MockClient(rm, map[string]string{
		"re.compile": "pattern",
	})

	assertExtract(t, src, rm, client, PathCounts{
		"re":               {Import: 1},
		"re.compile":       {Attribute: 1},
		"pattern":          {Expr: 1},
		"pattern.match.a":  {Attribute: 1},
		"pattern.search.b": {Attribute: 1},
		"foo":              {Import: 1, Name: 1},
	})
}

func TestLiterals(t *testing.T) {
	src := `
a = "hello"
a.upper()
b = {1:2}
b.update({3:4})
c = {1,2}
c.add(3)
d = [1,2,3]
d.append(4)
	`

	rm := pythonresource.MockManager(t, nil,
		"__builtin__.int", "__builtin__.str.upper", "__builtin__.dict.update", "__builtin__.set.add", "__builtin__.list.append")

	assertExtract(t, src, rm, nil, PathCounts{
		"__builtin__.str":         {Expr: 2},
		"__builtin__.str.upper":   {Attribute: 1},
		"__builtin__.dict":        {Expr: 2},
		"__builtin__.dict.update": {Attribute: 1},
		"__builtin__.set":         {Expr: 1},
		"__builtin__.set.add":     {Attribute: 1},
		"__builtin__.list":        {Expr: 1},
		"__builtin__.list.append": {Attribute: 1},
		"__builtin__.int":         {Expr: 11},
	})
}

func TestUnknownType(t *testing.T) {
	src := `
a = "hello"
a.upper()
a = NewThing()
a.thing()
`

	rm := pythonresource.MockManager(t, nil, "__builtin__.str.upper")

	assertExtract(t, src, rm, nil, PathCounts{
		"__builtin__.str":       {Expr: 2},
		"__builtin__.str.upper": {Attribute: 1},
		"__builtin__.str.thing": {Attribute: 1},
	})
}

func TestGlobalLiterals(t *testing.T) {
	src := `
a = "hello"
a.upper()
b = {1:2}
def test():
    b.update({3:4})
	`

	rm := pythonresource.MockManager(t, nil, "__builtin__.str.upper", "__builtin__.dict.update")

	assertExtract(t, src, rm, nil, PathCounts{
		"__builtin__.str":         {Expr: 2},
		"__builtin__.str.upper":   {Attribute: 1},
		"__builtin__.dict":        {Expr: 2},
		"__builtin__.dict.update": {Attribute: 1},
		"__builtin__.int":         {Expr: 4},
	})
}

func TestReturn(t *testing.T) {
	src := `
import string
def test1():
    return string.Template()

def test2():
    tmpl = string.Template()
    return tmpl.format()
`

	rm := pythonresource.MockManager(t, pythonresource.InfosFromKinds(map[string]pythonimports.Kind{
		"string.Template":        pythonimports.Type,
		"string.Template.format": pythonimports.Function,
	}))

	client := typeinduction.MockClient(rm, map[string]string{
		"string.Template": "string.Template",
	})

	assertExtract(t, src, rm, client, PathCounts{
		"string":                 {Import: 1},
		"string.Template":        {Attribute: 2, Expr: 2},
		"string.Template.format": {Attribute: 1},
	})
}

func TestClassMembers(t *testing.T) {
	src := `
import foo
import re

class Bar():
	def __init__(self, x):
		self.x = re.compile(x)
	def search(self):
		y = self.x.match()

class car(foo):
	def __init__(self, color="red"):
		self.vx = 0
		self.vy = 0
		self.color = color
	
	def accelerate(self, ax=0, ay=0, dt=0):
		self.vx += ax * dt
		self.vy += ay * dt

	def repaint(color):
		self.color = color
`

	rm := pythonresource.MockManager(t, nil, "foo")

	assertExtract(t, src, rm, nil, PathCounts{
		"foo":             {Import: 1, Name: 1},
		"__builtin__.str": {Expr: 1, Name: 2},
		"re.compile":      {Attribute: 1},
		"re":              {Import: 1},
		"__builtin__.int": {Expr: 7, Name: 7, Attribute: 2},
		"match":           {Attribute: 1},
	})
}

func TestExtractNoLocals(t *testing.T) {
	src := `
def zar():
    pass

g = zar()

class mass():
    pass

h = mass()
`
	rm := pythonresource.MockManager(t, nil)

	assertExtract(t, src, rm, nil, PathCounts{})
}

func TestAttrUnresolved(t *testing.T) {
	src := `
import foo
foo.bar
car.star
car.star.far.mar
`
	rm := pythonresource.MockManager(t, nil, "foo")

	assertExtract(t, src, rm, nil, PathCounts{
		"foo":              {Import: 1},
		"foo.bar":          {Attribute: 1},
		"car.star":         {Attribute: 1},
		"car.star.far.mar": {Attribute: 1},
	})
}

func TestImportsUnresolved(t *testing.T) {
	src := `
import foo.bar, bar.car
from foo import bar, star
	`
	rm := pythonresource.MockManager(t, nil)

	assertExtract(t, src, rm, nil, PathCounts{
		"foo.bar":  {Import: 2},
		"bar.car":  {Import: 1},
		"foo.star": {Import: 1},
	})
}
