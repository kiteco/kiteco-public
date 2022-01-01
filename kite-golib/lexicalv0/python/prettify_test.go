package python

import (
	"bytes"
	"strings"
	"testing"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/python"
)

var defaultConfig = Config{
	Indent:                      strings.Repeat(" ", 4),
	SpaceAfterColonInPair:       true,
	SpaceAfterColonInTypedParam: true,
	SpaceAfterColonInLambda:     true,
	SpaceAfterComma:             true,
	SpaceInfixOps:               true,
	SpaceAroundArrow:            true,
	BlankLinesBeforeClassDef:    2,
	BlankLinesBeforeTopFuncDef:  2,
	BlankLinesBetweenMethods:    1,
}

func TestPrettify_Imports(t *testing.T) {
	src := `
import spam
import numpy as np
from scipy import stats
from .data import Batch, DataFeeder, Dataset
`
	want := `
import spam
import numpy as np
from scipy import stats
from .data import Batch, DataFeeder, Dataset
`
	runPrettifyCase(t, defaultConfig, src, want)
}

func TestPrettify_Types(t *testing.T) {
	src := `
a = 42
b = "hello"
l = [3, 4, 5]
d = {a: b}
s = {1, 2, 3}
t = (1, 2, 3)
h = m[:, -1, :]
`
	want := `
a = 42
b = "hello"
l = [3, 4, 5]
d = {a: b}
s = {1, 2, 3}
t = (1, 2, 3)
h = m[:, -1, :]
`
	runPrettifyCase(t, defaultConfig, src, want)
}

func TestPrettify_Colon(t *testing.T) {
	src := `
a = [1:5]
b = {1: 5}
def foo(bar: Int):
    pass
`
	want := `
a = [1:5]
b = {1: 5}


def foo(bar: Int):
    pass
`
	runPrettifyCase(t, defaultConfig, src, want)
}

func TestPrettify_Operations(t *testing.T) {
	src := `
a = 3 + 5
b = "hello" + "world"
c = a * b
d = 42 % 6
e = not a
f = 5 and 4
g = a or b
num = 42 if a == 8 else 67
p = [
    o.k 
    for o in s 
    if o > 5
]
a = 5 + -5
`
	want := `
a = 3 + 5
b = "hello" + "world"
c = a * b
d = 42 % 6
e = not a
f = 5 and 4
g = a or b
num = 42 if a == 8 else 67
p = [o.k for o in s if o > 5]
a = 5 + -5
`
	runPrettifyCase(t, defaultConfig, src, want)
}

func TestPrettify_FunctionCalls(t *testing.T) {
	src := `
a = foo(bar, baz)
b = ', '.join(some_list)
accuracy, logits = inputs.sess.run([inputs.classifier.accuracy, inputs.classifier.logits], feed_dict=test_feed_dict)
`
	want := `
a = foo(bar, baz)
b = ', '.join(some_list)
accuracy, logits = inputs.sess.run([inputs.classifier.accuracy, inputs.classifier.logits], feed_dict=test_feed_dict)
`
	runPrettifyCase(t, defaultConfig, src, want)
}

func TestPrettify_Blocks(t *testing.T) {
	src := `
for i in range(0, 10):
    if i < 5:
        a = a + i
    elif i == 5:
        a = a + i * 2
		a += 1
    else:
        a = a - 1
    a = a + 5
print(a)
`
	want := `
for i in range(0, 10):
    if i < 5:
        a = a + i
    elif i == 5:
        a = a + i * 2
        a += 1
    else:
        a = a - 1
    a = a + 5
print(a)
`
	runPrettifyCase(t, defaultConfig, src, want)
}

func TestPretty_Definitions(t *testing.T) {
	src := `
import something


class ComplexNumber:
    def __init__(self,r: int = 0,i: int = 0, *vargs):
        self.real = r
        self.imag = i
    
    def getData(self):
        print("{0}+{1}j".format(self.real,self.imag))
    
    @staticmethod
    @something
    def wrap(naked: Callable[[Any], None]) -> Assertion:
        def assertion(_: str, x: Any):
            naked(x)
        return assertion
`
	want := `
import something


class ComplexNumber:
    def __init__(self, r: int=0, i: int=0, *vargs):
        self.real = r
        self.imag = i
    
    def getData(self):
        print("{0}+{1}j".format(self.real, self.imag))
    
    @staticmethod
    @something
    def wrap(naked: Callable[[Any], None]) -> Assertion:
        def assertion(_: str, x: Any):
            naked(x)
        return assertion
`
	runPrettifyCase(t, defaultConfig, src, want)
}

func TestPrettify_Lambda(t *testing.T) {
	src := `
a = (lambda x: x + 1)(2)
full_name = lambda first, last: f'Full name: {first.title()} {last.title()}'
`
	want := `
a = (lambda x: x + 1)(2)
full_name = lambda first, last: f'Full name: {first.title()} {last.title()}'
`
	runPrettifyCase(t, defaultConfig, src, want)
}

func TestPrettify_Try(t *testing.T) {
	src := `
try:
	f = open(arg, 'r')
except (OSError, RuntimeError):
	print('cannot open', arg)
else:
	print(arg, 'has', len(f.readlines()), 'lines')
	f.close()
`
	want := `
try:
    f = open(arg, 'r')
except (OSError, RuntimeError):
    print('cannot open', arg)
else:
    print(arg, 'has', len(f.readlines()), 'lines')
    f.close()
`
	runPrettifyCase(t, defaultConfig, src, want)
}

func TestPrettify_With(t *testing.T) {
	src := `
with tf.Session() as sess:
    tf.saved_model.loader.load(sess)
`
	want := `
with tf.Session() as sess:
    tf.saved_model.loader.load(sess)
`
	runPrettifyCase(t, defaultConfig, src, want)
}

func TestPrettify_Incomplete(t *testing.T) {
	src := `
src import numpy as np
from scipy import
`
	want := `
src import numpy as np
from scipy import
`
	runPrettifyCase(t, defaultConfig, src, want)
}

func TestPrettify_ErrorNode(t *testing.T) {
	src := `
class Some(Other):
    name = [string]
    def parse()
`
	want := `
class Some(Other):
    name = [string]
    def parse()
`
	runPrettifyCase(t, defaultConfig, src, want)
}

func runPrettifyCase(t *testing.T, conf Config, src, want string) {
	src, want = strings.TrimSpace(src), strings.TrimSpace(want)

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(python.GetLanguage())
	b := []byte(src)
	tree := parser.Parse(b)
	defer tree.Close()

	var buf bytes.Buffer
	if _, err := Prettify(&buf, conf, b, 0, len(b), tree.RootNode()); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != want {
		t.Fatalf("want:\n%s\ngot:\n%s\n", want, got)
	}
}
