package localcodetests

import "testing"

// TestClassDef tests there are no crash when a class is defined in a source file (issue #9047 on github)
func TestClassDef(t *testing.T) {
	classDef := `
import pandas
import collections

class MyClass(object):

    def __init__():
        self.blip = 22

df = {}
df["test"]= 5
	`

	assertResolveOpts(t, opts{
		src:     classDef,
		srcpath: "/code/classDef.py",
		localfiles: map[string]string{
			"/code/classDef.py": classDef,
		},
		expected: map[string]string{
			"df[\"test\"]": "5",
		},
	})
}
