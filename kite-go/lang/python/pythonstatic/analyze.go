package pythonstatic

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// AnalyzeGlobal analyzes source using only the global graph, in particular this means that
// the address assigned to symbols defined in the module is ARBITRARY.
func AnalyzeGlobal(ctx kitectx.Context, ai AssemblerInputs, ast *pythonast.Module) (*Assembly, error) {
	opts := DefaultOptions
	opts.AllowValueMutation = true
	assembler := NewAssembler(ctx, ai, opts)
	assembler.AddSource(ASTBundle{
		AST:     ast,
		Path:    "/src.py",
		Imports: FindImports(ctx, "/src.py", ast),
	})
	return assembler.Build(ctx)
}
