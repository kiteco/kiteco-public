package pythonproviders

import (
	"go/token"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/lexicalproviders"
	pythonlang "github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/applesilicon"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/python"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

const kitePlaceholder = "kite_placeholder_representation"

// Lexical ...
type Lexical struct{}

// Name implements Provider
func (Lexical) Name() data.ProviderName {
	return data.PythonLexicalProvider
}

// Provide implements Provider
func (p Lexical) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	// We do not support tensorflow models for Apple Silicon
	if applesilicon.Detected {
		return nil
	}

	// this check and IsSmart are set also at the lexicalcomplete:Python provider,
	// but do it here as well for clarity.
	_, isSmart := SmartProviders[p.Name()]
	if isSmart && g.Product.GetProduct() != licensing.Pro {
		return data.ProviderNotApplicableError{}
	}

	lexInputs, err := lexicalproviders.NewInputs(ctx, g.Lexical, in.SelectedBuffer, false)
	if err != nil {
		return err
	}
	analyzer, err := newSemanticAnalyzer(ctx, in.SelectedBuffer)
	if err != nil {
		return err
	}
	lexOut := func(c kitectx.Context, b data.SelectedBuffer, lexMc lexicalproviders.MetaCompletion) {
		filters := analyzer.filter(c, b, lexMc.Completion.Snippet)
		mc := MetaCompletion{
			Completion:         lexMc.Completion,
			Provider:           p.Name(),
			Source:             response.LexicalPythonSource,
			Score:              lexMc.Score,
			ExperimentalScore:  lexMc.ExperimentalScore,
			LexicalMeta:        &lexMc.LexicalMeta,
			LexicalMetrics:     lexMc.Metrics,
			LexicalFiltersMeta: &filters,
			FromSmartProvider:  isSmart,
			IsServer:           lexMc.IsServer,
		}
		out(c, b, mc)
	}
	return lexicalproviders.Text{}.Provide(ctx, g.Lexical, lexInputs, lexOut)
}

type semanticAnalyzer struct {
	definedNames map[string]bool
	prior        LexicalFiltersMeta
}

func newSemanticAnalyzer(c kitectx.Context, buf data.SelectedBuffer) (semanticAnalyzer, error) {
	analyzer := semanticAnalyzer{
		definedNames: make(map[string]bool),
	}
	pyLexer := python.Lexer{}
	tokens, err := pyLexer.Lex([]byte(buf.Buffer.Text()))
	if err != nil {
		return semanticAnalyzer{}, err
	}
	for _, token := range tokens {
		if pyLexer.IsType(lexer.IDENT, token) {
			analyzer.definedNames[token.Lit] = true
		}
	}
	// Initialize analyzer.prior based on the existing AST.
	// If e.g there is already an invalid name in the original AST,
	// then we won't filter invalid names later.
	analyzer.prior = analyzer.filter(c, buf, data.Snippet{})
	return analyzer, nil
}

func (s semanticAnalyzer) filter(c kitectx.Context, b data.SelectedBuffer, completion data.Snippet) LexicalFiltersMeta {
	var meta LexicalFiltersMeta
	completed := complete(b, represent(completion))
	ast := parse(c, completed)
	if ast == nil {
		return meta
	}
	imports := make(map[string]bool)
	classes := make(map[string]bool)

	pythonast.Inspect(
		ast,
		func(node pythonast.Node) bool {
			switch node := node.(type) {
			case *pythonast.Argument:
				// No new names used as arguments.
				// For example, in "foo(bar)", "bar" must not be a new name.
				if !s.prior.InvalidArgument && s.isNewName(node.Value) {
					meta.InvalidArgument = true
				}
			case *pythonast.AssignStmt:
				// No new names used for assignment.
				// For example, in "foo = bar", "bar" must not be a new name.
				if !s.prior.InvalidAssignment && s.isNewName(node.Value) {
					meta.InvalidAssignment = true
				}
			case *pythonast.AttributeExpr:
				// No attributes of new names.
				// For example, in "foo.bar", "foo" must not be a new name.
				if !s.prior.InvalidAttribute && s.isNewName(node.Value) {
					meta.InvalidAttribute = true
				}
			case *pythonast.BadStmt:
				// No bad statements.
				if !s.prior.HasBadStmt {
					meta.HasBadStmt = true
				}
			case *pythonast.ClassDefStmt:
				// No repeated class names.
				// For example, "class Foo:" can only appear once.
				if !s.prior.InvalidClassDef && classes[node.Name.Ident.Literal] {
					meta.InvalidClassDef = true
				}
				classes[node.Name.Ident.Literal] = true
			case *pythonast.DottedAsName:
				// No repeated imports.
				// For example, "import foo" can only appear once.
				if !s.prior.InvalidImport && imports[node.External.Join()] {
					meta.InvalidImport = true
				}
				imports[node.External.Join()] = true
			case *pythonast.FunctionDefStmt:
				// No repeated parameters in function definitions.
				// For example, "def foo(bar, bar):" has repeated parameters.
				if !s.prior.InvalidFunctionDef && repeatedParams(node.Parameters) {
					meta.InvalidFunctionDef = true
				}
			case *pythonast.ImportAsName:
				// No repeated imports.
				// For example, "from foo import bar" can only appear once.
				if !s.prior.InvalidFunctionDef && imports[node.External.Ident.Literal] {
					meta.InvalidImport = true
				}
				imports[node.External.Ident.Literal] = true
			}
			return true
		},
	)
	return meta
}

func (s semanticAnalyzer) isNewName(expr pythonast.Expr) bool {
	nameExpr, isNameExpr := expr.(*pythonast.NameExpr)
	if !isNameExpr {
		return false
	}
	if nameExpr == nil {
		return false
	}
	if s.definedNames[nameExpr.Ident.Literal] {
		return false
	}
	if _, ok := pythonlang.Builtins[nameExpr.Ident.Literal]; ok {
		return false
	}
	if nameExpr.Ident.Literal == kitePlaceholder {
		return false
	}
	return true
}

func represent(completion data.Snippet) string {
	var parts []string
	completion.Iterate(func(text string, ph bool) bool {
		if text == "" {
			return true
		}
		if ph {
			parts = append(parts, kitePlaceholder)
			return true
		}
		parts = append(parts, text)
		return true
	})
	return strings.Join(parts, "")
}

func complete(given data.SelectedBuffer, completion string) data.SelectedBuffer {
	// Determines what the buffer would be if `completion` is inserted into `given`
	size := lexicalproviders.OverlapSize(given, completion)
	overlap := data.Selection{
		Begin: given.Selection.Begin - size,
		End:   given.Selection.End,
	}
	return given.Buffer.Select(overlap).ReplaceWithCursor(completion)
}

func parse(c kitectx.Context, buffer data.SelectedBuffer) *pythonast.Module {
	cursor := token.Pos(buffer.Selection.Begin)
	parseOpts := pythonparser.Options{
		Approximate: true,
		Cursor:      &cursor,
		ScanOptions: pythonscanner.Options{
			ScanComments: true,
			ScanNewLines: true,
		},
	}
	module, _ := pythonparser.Parse(c, []byte(buffer.Buffer), parseOpts)
	return module
}

func repeatedParams(params []*pythonast.Parameter) bool {
	seen := make(map[string]bool)
	for _, param := range params {
		nameExpr, isNameExpr := param.Name.(*pythonast.NameExpr)
		if !isNameExpr {
			continue
		}
		if seen[nameExpr.Ident.Literal] {
			return true
		}
		seen[nameExpr.Ident.Literal] = true
	}
	return false
}
