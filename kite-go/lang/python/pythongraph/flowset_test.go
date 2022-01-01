package pythongraph

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
	"github.com/stretchr/testify/require"
)

type flowsToCase struct {
	name     string
	idx      int
	expected []int
}

func newCase(name string, idx int, expected ...int) flowsToCase {
	return flowsToCase{
		name:     name,
		idx:      idx,
		expected: expected,
	}
}

func expectedNames(ns *nameSet, expectedIdxs []int) *nameSet {
	expected := newNameSet()
	ordered := ns.Names()

	for _, ei := range expectedIdxs {
		name := ordered[ei]
		expected.Add(name, ns.Set()[name])
	}

	return expected
}

func namesString(lm *linenumber.Map, src string, ns *nameSet) string {
	var names []*pythonast.NameExpr

	for n := range ns.Set() {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		ni, nj := names[i], names[j]
		return ns.Set()[ni] < ns.Set()[nj]
	})

	var parts []string
	for _, name := range names {
		line, cols := lm.LineCol(int(name.Begin()))
		cole := lm.Column(int(name.End()))
		lb, le := lm.LineBounds(line)
		parts = append(parts, fmt.Sprintf("%d[%d:%d][%d:%d] %s\nLine: %s",
			line, cols, cole, name.Begin(), name.End(),
			pythonast.String(name), src[lb:le]))
	}
	return strings.Join(parts, "\n")
}

func requireNamesForTest(t *testing.T, name string, b *graphBuilder) *nameSet {
	var names *nameSet
	for _, v := range b.vm.Variables {
		ns := v.Refs
		var matches bool
		for ne := range ns.Set() {
			if ne.Ident.Literal == name {
				matches = true
			}
			break
		}
		if matches {
			names = ns
			break
		}
	}

	require.NotNil(t, names, "unable to find symbols for name %s", name)
	return names
}

func diffNameSets(expected, actual *nameSet) (*nameSet, *nameSet) {
	missing := newNameSet()
	for e, order := range expected.Set() {
		if _, ok := actual.Get(e); !ok {
			missing.Add(e, order)
		}
	}

	extra := newNameSet()
	if !actual.Empty() {
		for a, order := range actual.Set() {
			if _, ok := expected.Get(a); !ok {
				extra.Add(a, order)
			}
		}
	}
	return missing, extra
}

func assertFlowsTo(t *testing.T, src string, cases ...flowsToCase) {
	src = strings.TrimSpace(src)

	b := requireBuilder(t, emptyRM(t), src)

	for i, c := range cases {

		names := requireNamesForTest(t, c.name, b)

		expected := expectedNames(names, c.expected)

		graph := b.forwardFlowGraph(names)

		name := names.Names()[c.idx]
		actual := graph[name]

		lm := linenumber.NewMap([]byte(src))

		fmtstr := `
------------- Case %d --------------
Src:
%s
Name: %d[%d:%d][%d:%d]: %s
Line: %s
	`

		var srcWithLineNumbers string
		for i, line := range strings.Split(src, "\n") {
			srcWithLineNumbers += fmt.Sprintf("%d: %s\n", i, line)
		}

		line := lm.Line(int(name.Begin()))
		cb, ce := lm.Column(int(name.Begin())), lm.Column(int(name.End()))
		lb, le := lm.LineBounds(line)

		errorf := func(extrafmt string, args ...interface{}) {
			fmtstr = fmtstr + extrafmt
			all := []interface{}{
				i,
				srcWithLineNumbers,
				line, cb, ce, name.Begin(), name.End(),
				c.name,
				src[lb:le],
			}
			all = append(all, args...)
			t.Errorf(fmtstr, all...)
		}

		missing, extra := diffNameSets(expected, actual)

		switch {
		case extra.Len() > 0 && missing.Len() > 0:
			fmtstr := `
Got %d extra:
%s
Got %d missing:
%s
`
			errorf(fmtstr, extra.Len(), namesString(lm, src, extra), missing.Len(), namesString(lm, src, missing))
		case extra.Len() > 0:
			fmtstr := `
Got %d extra:
%s
`
			errorf(fmtstr, extra.Len(), namesString(lm, src, extra))
		case missing.Len() > 0:
			fmtstr := `
Got %d missing:
%s
`
			errorf(fmtstr, missing.Len(), namesString(lm, src, missing))
		}
	}
}

type entryOrExitCase struct {
	name     string
	expected []int
	idxStmt  int
	entry    bool
}

func newEntryOrExitCase(name string, idxStmt int, entry bool, expected ...int) entryOrExitCase {
	return entryOrExitCase{
		name:     name,
		expected: expected,
		idxStmt:  idxStmt,
		entry:    entry,
	}
}

func assertEntryOrExitSet(t *testing.T, src string, cases ...entryOrExitCase) {
	src = strings.TrimSpace(src)

	b := requireBuilder(t, emptyRM(t), src)

	for i, c := range cases {

		ns := requireNamesForTest(t, c.name, b)

		stmt := b.a.RAST.Root.Body[c.idxStmt]

		gb := newNameGraphBuilder(ns, b.a.RAST.Root)

		actual := newNameSet()
		if c.entry {
			gb.addEntrySet(actual, stmt)
		} else {
			gb.addExitSet(actual, stmt)
		}

		expected := expectedNames(ns, c.expected)

		lm := linenumber.NewMap([]byte(src))

		fmtstr := `
------------- Case %d --------------
Src:
%s
Name: %s
%s Set for Stmt:
%s
	`

		var srcWithLineNumbers string
		for i, line := range strings.Split(src, "\n") {
			srcWithLineNumbers += fmt.Sprintf("%d: %s\n", i, line)
		}

		errorf := func(extrafmt string, args ...interface{}) {
			ty := "Exit"
			if c.entry {
				ty = "Entry"
			}
			fmtstr = fmtstr + extrafmt
			all := []interface{}{
				i,
				srcWithLineNumbers,
				c.name,
				ty,
				printNode(stmt),
			}
			all = append(all, args...)
			t.Errorf(fmtstr, all...)
		}

		missing, extra := diffNameSets(expected, actual)

		switch {
		case extra.Len() > 0 && missing.Len() > 0:
			fmtstr := `
Got %d extra:
%s
Got %d missing:
%s
`
			errorf(fmtstr, extra.Len(), namesString(lm, src, extra), missing.Len(), namesString(lm, src, missing))
		case extra.Len() > 0:
			fmtstr := `
Got %d extra:
%s
`
			errorf(fmtstr, extra.Len(), namesString(lm, src, extra))
		case missing.Len() > 0:
			fmtstr := `
Got %d missing:
%s
`
			errorf(fmtstr, missing.Len(), namesString(lm, src, missing))
		}
	}
}

func TestFlowsToSequential(t *testing.T) {
	src := `
x = 1
x = x + 1
y = 1
y += y + 1
foo(x)
x
`

	assertFlowsTo(t, src,
		newCase("x", 0, 1),
		newCase("x", 1, 2),
		newCase("x", 2, 3),
		newCase("x", 3, 4),
		newCase("x", 4),
		newCase("y", 0, 1),
		newCase("y", 1, 2),
		newCase("y", 2),
	)
}

func TestEntryAndExitSetForLoop(t *testing.T) {
	src := `
x = 1
for x in [x]:
	break
	x = x - 1
else:
	x = x + 1

y = 1
for y in []:
	y
	break
else:
	y
`

	assertEntryOrExitSet(t, src,
		newEntryOrExitCase("x", 1, true, 1),
		newEntryOrExitCase("x", 1, false, 1, 2, 4, 6),
		newEntryOrExitCase("y", 3, true, 1, 3),
		newEntryOrExitCase("y", 3, false, 1, 2, 3),
	)
}

func TestFlowsToForLoopSimple(t *testing.T) {
	src := `
x = 1
for x in [x]:
	break
	x = x + 1
else:
	x = x - 1

x
`

	assertFlowsTo(t, src,
		newCase("x", 0, 1),
		newCase("x", 1, 2, 5, 7),
		newCase("x", 2, 3, 7),
		newCase("x", 3, 4),
		newCase("x", 4, 1, 5, 7),
		newCase("x", 5, 6),
		newCase("x", 6, 7),
		newCase("x", 7),
	)
}

func TestFlowsToForLoopNoTargetsOrIters(t *testing.T) {
	src := `
x = 1
for _ in L:
	x = 1
	x
else:
	x
x
`
	assertFlowsTo(t, src,
		newCase("x", 0, 1, 3, 4),
		newCase("x", 1, 2),
		newCase("x", 2, 1, 3, 4),
	)
}

func TestFlowsToForLoopSelfIters(t *testing.T) {
	src := `
x = 1
for _ in [x]:
	pass
	`

	assertFlowsTo(t, src,
		newCase("x", 0, 1),
		newCase("x", 1, 1),
	)
}

func TestFlowsToForLoopSelfTargets(t *testing.T) {
	src := `
x = 1
for x in []:
	pass
	`
	assertFlowsTo(t, src,
		newCase("x", 0, 1),
		newCase("x", 1, 1),
	)
}

func TestEntryAndExitSetWhileLoop(t *testing.T) {
	src := `
x = 1
while x > 0:
	x = x - 1
else:
	x = x + 1

y = 1
while True:
	y = y - 1
else:
	y = y + 1
`
	assertEntryOrExitSet(t, src,
		newEntryOrExitCase("x", 1, true, 1),
		newEntryOrExitCase("x", 1, false, 1, 3, 5),
		newEntryOrExitCase("y", 3, true, 1, 3),
		newEntryOrExitCase("y", 3, false, 2, 4),
	)
}

func TestFlowsToWhileLoop(t *testing.T) {
	src := `
x = 1
while x > 0:
	x = x + 1
	if something():
		break
else:
	x = x - 1

x
`

	assertFlowsTo(t, src,
		newCase("x", 0, 1),
		newCase("x", 1, 2, 4, 6),
		newCase("x", 2, 3),
		newCase("x", 3, 4, 1, 6),
		newCase("x", 4, 5),
		newCase("x", 5, 6),
		newCase("x", 6),
	)

}

func TestFlowsToWhileLoopEmptyBody(t *testing.T) {
	src := `
x = 1
while x:
	pass

x
	`

	assertFlowsTo(t, src,
		newCase("x", 0, 1),
		newCase("x", 1, 1, 2),
		newCase("x", 2),
	)
}

func TestFlowsToWhileLoopNotInCondition(t *testing.T) {
	src := `
x = 1
while foo():
	if maybeBreak():
		break
	x = x + 1
else:
	x = x - 1
x
`

	assertFlowsTo(t, src,
		newCase("x", 0, 1, 3, 5),
		newCase("x", 1, 2),
		newCase("x", 2, 1, 3, 5),
		newCase("x", 3, 4),
		newCase("x", 4, 5),
		newCase("x", 5),
	)
}

func TestEntryAndExitSetIf(t *testing.T) {
	src := `
x = 2
if foo():
	x = x + 1
elif x > 1:
	if False:
		x = x + 2
elif x > 0:
	x = x + 3
else:
	x = x + 4

y = 1
if foo():
	if False:
		y = y + 1
else:
	y = y + 2
`

	assertEntryOrExitSet(t, src,
		newEntryOrExitCase("x", 1, true, 1, 3),
		newEntryOrExitCase("x", 1, false, 2, 3, 5, 6, 8, 10),
		newEntryOrExitCase("y", 3, true, 1, 3),
		newEntryOrExitCase("y", 3, false, 2, 4),
	)
}

func TestFlowsToIfBasic(t *testing.T) {
	src := `
x = 2
if foo():
	if False:
		x = x + 1
elif x > 1:
	if False:
		x = x + 2
elif x > 0:
	x = x + 3
else:
	x = x + 4

x
`

	assertFlowsTo(t, src,
		newCase("x", 0, 1, 3, 11),
		newCase("x", 1, 2),
		newCase("x", 2, 11),
		newCase("x", 3, 4, 6, 11),
		newCase("x", 4, 5),
		newCase("x", 5, 11),
		newCase("x", 6, 7, 9, 11),
		newCase("x", 7, 8),
		newCase("x", 8, 11),
		newCase("x", 9, 10),
		newCase("x", 10, 11),
		newCase("x", 11),
	)

}

func TestFlowsToIfNoCondition(t *testing.T) {
	src := `
x = 1
if foo():
	x = x + 1
else:
	x = x + 2

x
`

	assertFlowsTo(t, src,
		newCase("x", 0, 1, 3, 5),
		newCase("x", 1, 2),
		newCase("x", 2, 5),
		newCase("x", 3, 4),
		newCase("x", 4, 5),
		newCase("x", 5),
	)

}

func TestFlowsToIfExpr(t *testing.T) {
	src := `
x = 1
x = x if x > 1 else x - 1
`
	assertFlowsTo(t, src,
		newCase("x", 0, 1, 2, 3),
		newCase("x", 1, 4),
		newCase("x", 2, 4),
		newCase("x", 3, 4),
		newCase("x", 4),
	)
}

func TestFlowsToLoopAndConditional(t *testing.T) {
	src := `
for x in L:
	if x > 0:
		x  = x + 1
	elif x < -1:
		x = x -1
	else:
		x = x +2
`

	assertFlowsTo(t, src,
		newCase("x", 0, 1),
		newCase("x", 1, 2, 4, 0),
		newCase("x", 2, 3),
		newCase("x", 3, 0),
		newCase("x", 4, 5, 7, 0),
		newCase("x", 5, 6),
		newCase("x", 6, 0),
		newCase("x", 7, 8),
		newCase("x", 8, 0),
	)

}

func TestFlowsToFunction(t *testing.T) {
	src := `
def foo(x,y):
	x = 1
	y = 2
	`

	assertFlowsTo(t, src,
		newCase("x", 0, 1),
		newCase("x", 1),
		newCase("y", 0, 1),
		newCase("y", 1),
	)
}

func TestFlowsToWithStmt(t *testing.T) {
	src := `
opener = 1
anotherOpener = 2
ff = 1
open = 1
with opener() as f, open() as ff:
	f
	anotherOpener
anotherOpener
opener
f
ff
	`

	assertFlowsTo(t, src,
		newCase("opener", 0, 1),
		newCase("opener", 1, 2),
		newCase("anotherOpener", 0, 1, 2),
		newCase("anotherOpener", 1, 2),
		newCase("ff", 0, 1, 2),
		newCase("ff", 1, 2),
	)

}

func TestFlowsToImportName(t *testing.T) {
	src := `
import numpy as np, numpy.foo as foo, os.path

np

foo

os
	`

	assertFlowsTo(t, src,
		newCase("np", 0, 1),
		newCase("foo", 0, 1),
		newCase("os", 0, 1),
	)
}

func TestFlowsToImportFrom(t *testing.T) {
	src := `
from matplotlib import pyplot, plotly

pyplot.plot()

plotly()

matplotlib
	`

	assertFlowsTo(t, src,
		newCase("pyplot", 0, 1),
		newCase("plotly", 0, 1),
		newCase("matplotlib", 0),
	)
}

func TestBlocksFlowClassAndFunc(t *testing.T) {
	src := `
x = 1

def foo():
	pass

x

y = 1

class foo():
	pass

y
	`

	assertFlowsTo(t, src,
		newCase("x", 0, 1),
		newCase("y", 0, 1),
	)
}

func TestFlowsToFunctionDefAndClassDef(t *testing.T) {
	t.Skip("TODO: add flow set to function")
	src := `
def foo(z, (q, (v,w))):
	z
	q
	v
	w
	pass

foo

x = 1
class bar():
	pass
	def foo():
		x

bar
	`

	assertFlowsTo(t, src,
		newCase("foo", 0, 1),
		newCase("z", 0, 1),
		newCase("q", 0, 1),
		newCase("v", 0, 1),
		newCase("x", 0, 1),
		newCase("bar", 0, 1),
	)
}

func TestFlowsToTryStatement(t *testing.T) {
	src := `
x = 1
try:
	x = 2
except Exception as e:
	e
`

	assertFlowsTo(t, src,
		newCase("x", 0, 1),
		newCase("e", 0, 1),
	)
}
