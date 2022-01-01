package pythongraph

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// variable links together multiple name expression nodes in the graph that all correspond to
// the same variable (symbol), see README
type variable struct {
	Origin *pythonast.NameExpr
	First  int
	ID     VariableID
	Refs   *nameSet
}

type scope []*variable

func (s scope) String() string {
	var vs []string
	for _, v := range s {
		vs = append(vs, v.String())
	}

	return fmt.Sprintf("{%s}", strings.Join(vs, ", "))
}

func (s scope) Contains(v *variable) bool {
	for _, vs := range s {
		if v == vs {
			return true
		}
	}
	return false
}

func (v *variable) String() string {
	if v == nil {
		return "nil"
	}
	ns := fmt.Sprintf("{%s %d %d}", v.Origin.Ident.Literal, v.Origin.Begin(), v.Origin.End())
	return fmt.Sprintf("{Origin: %s, First: %d, ID: %d, Refs: %v}", ns, v.First, v.ID, v.Refs)
}

type nameToVariable map[*pythonast.NameExpr]*variable

type variableManager struct {
	Variables      []*variable
	nameToVariable nameToVariable
	tree           *variableTreeNode
}

func newVariableManager(a *analysis, addMissingNames bool) *variableManager {
	symbolToNames := make(map[*pythontype.Symbol]*nameSet)
	var numNames int

	addName := func(name *pythonast.NameExpr) {
		table := a.TableForName(name)
		if table == nil {
			// TODO(juan): should not happen?
			log.Println("no table for", name.Ident.Literal)
			return
		}

		sym := table.Find(name.Ident.Literal)
		if sym == nil {
			if addMissingNames && !okToIgnoreMissingName(name, a.RAST.Parent) {
				// add the name to the first symbol table in which is can be resolved,
				// this can happen if the user references the name of a variable before he
				// has initialized it in scope, e.g consider
				// `print(x)` in this case x will never have been assigned to a symbol before it is referenced.
				// TODO(juan): clean this up, modifying the resolved ast is not great, but unclear how we would
				// do this without duplicating all of the symbol tables.

				sym = table.Create(name.Ident.Literal)
				a.RAST.References[name] = nil
			} else {
				return
			}
		}

		ns := symbolToNames[sym]
		if ns == nil {
			ns = newNameSet()
			symbolToNames[sym] = ns
		}

		numNames++

		ns.Add(name, a.RAST.Order[name])
	}

	pythonast.Inspect(a.RAST.Root, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}

		// have to handle imports manually because we use
		// name expressions in the ast when we should not
		switch node := node.(type) {
		case *pythonast.ImportFromStmt:
			for _, clause := range node.Names {
				if clause.Internal != nil {
					addName(clause.Internal)
				} else if clause.External != nil {
					addName(clause.External)
				}
			}
			return false
		case *pythonast.ImportNameStmt:
			for _, clause := range node.Names {
				if clause.Internal != nil {
					addName(clause.Internal)
				} else if len(clause.External.Names) > 0 {
					addName(clause.External.Names[0])
				}
			}
			return false
		}

		name, ok := node.(*pythonast.NameExpr)
		if !ok {
			return true
		}

		addName(name)

		return false
	})

	symbols := make([]*pythontype.Symbol, 0, len(symbolToNames))
	for sym := range symbolToNames {
		symbols = append(symbols, sym)
	}

	sort.Slice(symbols, func(i, j int) bool {
		nsi, nsj := symbolToNames[symbols[i]], symbolToNames[symbols[j]]

		iOrigin, jOrigin := nsi.Names()[0], nsj.Names()[0]

		iOrder, jOrder := nsi.Set()[iOrigin], nsj.Set()[jOrigin]

		return iOrder < jOrder
	})

	variables := make([]*variable, 0, len(symbols))
	nameToVariable := make(nameToVariable, numNames)
	for _, sym := range symbols {
		ns := symbolToNames[sym]
		origin := ns.Names()[0]
		v := &variable{
			ID:     VariableID(len(variables)),
			Origin: origin,
			First:  ns.Set()[origin],
			Refs:   ns,
		}
		variables = append(variables, v)

		for name := range ns.Set() {
			nameToVariable[name] = v
		}
	}

	return &variableManager{
		Variables:      variables,
		nameToVariable: nameToVariable,
		tree:           newVariableTree(a.RAST.Root, nameToVariable),
	}
}

func (vm *variableManager) InScope(at pythonast.Node, stopAtFunc bool) scope {
	return vm.tree.InScope(at, stopAtFunc)
}

func (vm *variableManager) VariableText(i VariableID) string {
	return vm.Variables[i].Origin.Ident.Literal
}

func (vm *variableManager) VariableFor(name *pythonast.NameExpr) *variable {
	return vm.nameToVariable[name]
}

func (vm *variableManager) VariableIDFor(name *pythonast.NameExpr) VariableID {
	v := vm.VariableFor(name)
	if v == nil {
		return -1
	}
	return v.ID
}

// ReduceTo removes all variables that are not in scope
// and renumbers the variables in scope have ids in {0,1,...,len(scope)}
// TODO: expensive, can we optimize this? Or avoid this?
func (vm *variableManager) ReduceTo(root *pythonast.Module, scope scope) {
	// sort scope by appearance in ast to make things consistent
	sort.Slice(scope, func(i, j int) bool {
		return scope[i].First < scope[j].First
	})

	// renumber
	nameToVariable := make(nameToVariable)
	for i, v := range scope {
		v.ID = VariableID(i)
		for _, ref := range v.Refs.Names() {
			nameToVariable[ref] = v
		}
	}

	vm.Variables = scope
	vm.nameToVariable = nameToVariable
	vm.tree = newVariableTree(root, nameToVariable)
}

func sortVars(vs []*variable) {
	sort.Slice(vs, func(i, j int) bool {
		return vs[i].First < vs[j].First
	})
}

func okToIgnoreMissingName(name *pythonast.NameExpr, parents map[pythonast.Node]pythonast.Node) bool {
	for parent := parents[name]; parent != nil; parent = parents[parent] {
		switch t := parent.(type) {
		case *pythonast.Argument:
			return name == t.Name
		case *pythonast.DottedExpr:
			// This can occur for
			// 1) the package portion of an import from statement,
			// 2) the external portion of a clause in an import name statement
			// For 1) these names are never added to the scope so it is safe to ignore.
			// For 2) we should technically be checking if the name of interest
			// is the root of the dotted expression and the internal clause of the
			// DottedAsName is nil, in which case we should return false.
			// For now we ignore the added complexity of case 2, we can revisit this later.
			// TODO(juan): revisit
			return true
		case *pythonast.ImportAsName:
			nameInScope := t.External
			if t.Internal != nil {
				nameInScope = t.Internal
			}
			return name != nameInScope
		}
	}
	return false
}

// CountNumVars counts number of valid arguments that can be passed to a call expression
func CountNumVars(b []byte, words []pythonscanner.Word, rast *pythonanalyzer.ResolvedAST, node pythonast.Node,
	rm pythonresource.Manager) int {
	a := newAnalysis(rm, words, rast)

	ctx := kitectx.Background()
	vm := newVariableManager(a, false)

	scope := vm.InScope(node, true)

	scope = reduceScopeForCall(ctx, a, scope)

	return len(scope)
}
