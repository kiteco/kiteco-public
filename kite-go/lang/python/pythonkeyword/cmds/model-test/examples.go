package main

import s "github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"

type example struct {
	src string
	// Either token or tokens should be set
	token  s.Token
	tokens []s.Token
}

var (
	examples []example
)

func init() {
	examples = []example{
		// Names
		{src: `for foo in `, token: s.Ident},
		{src: `try:
	foo = bar
except `, token: s.Ident},
		{src: `def foo(bar=`, token: s.Ident},
		{src: `foo = `, token: s.Ident},
		{src: `foo = b`, token: s.Ident},
		{src: `foo(`, token: s.Ident},
		{src: `foo(bar, `, token: s.Ident},
		{src: `foo = {"bar": `, token: s.Ident},
		{src: `foo = {`, token: s.Ident},
		{src: `foo = [`, token: s.Ident},
		{src: `foo = (bar, `, token: s.Ident},

		// Keywords
		{src: `i`, token: s.Import},
		{src: `f`, token: s.From},

		{src: `import baz

def bar():
	for foo `, token: s.In},
		{src: `import baz

def bar():
	for foo i`, token: s.In},

		{src: `import baz

def bar():
	if foo == bar a`, token: s.And},
		{src: `import baz

def bar():
	if foo == bar o`, token: s.Or},
		{src: `import baz

def bar():
	if foo i`, tokens: []s.Token{s.In, s.Is}},
		{src: `import baz

def bar():
	if foo n`, token: s.Not},
		{src: `import baz
 
def bar():
	if foo is n`, token: s.Not},

		{
			src: `class Bar(object):
	d`,
			token: s.Def,
		},
		{src: `d`, token: s.Def},
		{src: `c`, token: s.Class},
		{
			src: `import baz

def bar():
	for foo in range(10):
		if foo == bar:
			b`,
			token: s.Break, // harder
		},
		{
			src: `import baz

def bar():
	for foo in range(10):
		if foo == bar:
			c`,
			token: s.Continue, // harder
		},
		{
			src: `import baz

def bar():
	if foo:
		bar = baz
	e`,
			tokens: []s.Token{s.Else, s.Elif},
		},
		{
			src: `try:
	bar = baz
e`,
			token: s.Except,
		},
		{
			src: `try:
	bar = baz
except foo:
	bar = baz
e`,
			token: s.Except,
		},
		{
			src: `import foo

def bar():
	try:
		bar = baz
	f`,
			token: s.Finally,
		},
		{
			src: `class Foo:
	p`,
			token: s.Pass, // harder
		},
		{
			src: `def foo():
	r`,
			token: s.Return, // harder
		},
		{
			src: `for foo in range(bar):
	y`,
			token: s.Yield, // harder
		},
		{src: `import baz

def bar():
	myVar = 5
	return r`, token: s.Ident},

		{src: `import os.path
from requests import get


get_alias = get


def duplicate_function():
    x = "test"
    y = x.join(["foo", "bar", "baz"])
    (z for z in y if z != x)
    f = lambda x: x
    f(1)


def duplicate_function(var):
    return os.path.join("/foo", "bar/")


duplicate_function_alias = duplicate_function
result = duplicate_function_alias(None)
result = duplicate_function_alias()
p`, token: s.Ident},
		{src: `import os.path
from requests import get


get_alias = get


def duplicate_function():
    x = "test"
    y = x.join(["foo", "bar", "baz"])
    (z for z in y if z != x)


def duplicate_function(var):
    return os.path.join("/foo", "bar/")


duplicate_function_alias = duplicate_function
result = duplicate_function_alias(None)
result = duplicate_function_alias()
p`, token: s.Ident},
	}
	// Useful line to debug examples :
	//examples = examples[len(examples)-2:]
}
