package localcodetests

import "testing"

func TestParamTypes(t *testing.T) {
	src := `
def foo(a, b, c):
	out1 = a
	out2 = b
	out3 = c

foo(1, "xyz", 0.5)
`
	assertResolveOpts(t, opts{
		src:     src,
		srcpath: "/code/src.py",
		expected: map[string]string{
			"out1": "instanceof builtins.int",
			"out2": "instanceof builtins.str",
			"out3": "instanceof builtins.float",
		},
	})
}
