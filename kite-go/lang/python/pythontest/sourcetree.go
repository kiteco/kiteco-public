package pythontest

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// SourceTree with the specified files
// files: map from path name for the module as it should appear in the source tree to the path for the file.
func SourceTree(uid int64, mid string, graph pythonresource.Manager, files map[string]string) (*pythonenv.SourceTree, error) {
	ai := pythonstatic.AssemblerInputs{
		User:    uid,
		Machine: mid,
		Graph:   graph,
	}
	opts := pythonstatic.DefaultOptions
	opts.AllowValueMutation = true
	assembler := pythonstatic.NewAssembler(kitectx.Background(), ai, pythonstatic.DefaultOptions)
	for path, truepath := range files {
		buf, err := ioutil.ReadFile(truepath)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %v", truepath, err)
		}

		parseopts := pythonparser.Options{
			ErrorMode: pythonparser.FailFast,
		}

		mod, err := pythonparser.Parse(kitectx.Background(), buf, parseopts)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %v", truepath, err)
		}

		assembler.AddSource(pythonstatic.ASTBundle{AST: mod, Path: path, Imports: pythonstatic.FindImports(kitectx.Background(), path, mod)})
	}

	var assembly *pythonstatic.Assembly
	var ft *pythonenv.FlatSourceTree
	err := kitectx.Background().WithTimeout(10*time.Second, func(ctx kitectx.Context) error {
		var err error

		assembly, err = assembler.Build(ctx)
		if err != nil {
			return fmt.Errorf("error analyzing files: %v", err)
		}

		ft, err = assembly.Sources.Flatten(ctx)
		if err != nil {
			return fmt.Errorf("error flattending source tree: %v", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	st, err := ft.Inflate(graph)
	if err != nil {
		return nil, fmt.Errorf("error inflating source tree: %v", err)
	}

	return st, nil
}
