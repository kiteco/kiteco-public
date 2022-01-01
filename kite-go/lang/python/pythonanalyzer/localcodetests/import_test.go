package localcodetests

import "testing"

func TestLocalCodeImports(t *testing.T) {
	fooSrc := `abc = 123`
	barSrc := `import foo; out = foo.abc`
	assertResolveOpts(t, opts{
		src:     barSrc,
		srcpath: "/code/bar.py",
		localfiles: map[string]string{
			"/code/foo.py": fooSrc,
		},
		expected: map[string]string{
			"out": "instanceof builtins.int",
		},
	})
}

func TestLocalCodeSimple(t *testing.T) {
	math := `
e = exp(1)

def power(x, n):
	val = 1.0
	for i in range(n):
		val *= x
	if n < 0:
		return 1.0 / val
	return val

def factorial(n):
	if n < 2:
		return 1
	return n * factorial(n - 1)

def exp(x):
	val = 0.0
	for i in range(100):
		val += power(x, i) / factorial(i)
	return val
	`

	assertResolveOpts(t, opts{
		src:     math,
		srcpath: "/code/math.py",
		localfiles: map[string]string{
			"/code/math.py": math,
		},
		expected: map[string]string{
			"e": "instanceof (generic:builtins.float | generic:builtins.int)",
		},
	})
}
