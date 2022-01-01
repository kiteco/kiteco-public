package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

var (
	largeFileThreshold = int(1e5) // files over this size (in bytes) may be noise
	itersDataProp      = 3

	countCmd = cmdline.Command{
		Name:     "count",
		Synopsis: "Counting missing modules and attributes",
		Args:     &countArgs{},
	}
)

type counts struct {
	MissingModules     map[string]int64
	MissingModuleAttrs map[string]int64
	MissingClassAttrs  map[string]map[string]int64
}

func newCounts() *counts {
	return &counts{
		MissingModules:     make(map[string]int64),
		MissingModuleAttrs: make(map[string]int64),
		MissingClassAttrs:  make(map[string]map[string]int64),
	}
}

func (counts *counts) countImports(graph *pythonimports.Graph, sources *pythonstatic.SourceTree, file *pythonstatic.File) map[string]string {
	// internal name -> external name (path)
	imported := make(map[string]string)
	pythonast.Inspect(file.ASTBundle.AST, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}
		switch node := node.(type) {
		case pythonast.Expr:
			return false
		case *pythonast.ImportNameStmt:
			for _, clause := range node.Names {
				if clause == nil || clause.External == nil || len(clause.External.Names) == 0 {
					continue
				}

				// try local graph
				if symb := sources.ImportAbs(file.ASTBundle.Path, clause.External.Names[0].Ident.Literal); symb != nil {
					// TODO(juan): user could still reference a global package here but lets ignore for now
					continue
				}

				// try global graph
				path := clause.External.Join()
				if mod, err := graph.Navigate(pythonimports.NewDottedPath(path)); mod == nil || err != nil {
					counts.MissingModules[path]++
					if clause.Internal != nil {
						imported[clause.Internal.Ident.Literal] = path
					} else {
						imported[path] = path
					}
				}
			}
			return false
		case *pythonast.ImportFromStmt:
			if node.Package == nil || len(node.Dots) > 0 || len(node.Package.Names) == 0 {
				// TODO(juan): user could still reference a global package for the last 2 clauses,
				// but lets ignore for now
				return false
			}

			// try local graph
			if symb := sources.ImportAbs(file.ASTBundle.Path, node.Package.Names[0].Ident.Literal); symb != nil {
				// TODO(juan): user could still reference a global package from here, but lets ignore for now
				return false
			}

			// try global graph
			base := node.Package.Join()
			baseNode, _ := graph.Navigate(pythonimports.NewDottedPath(base))
			if baseNode == nil {
				counts.MissingModules[base]++
				for _, clause := range node.Names {
					if clause == nil || clause.External == nil {
						continue
					}
					path := base + "." + clause.External.Ident.Literal
					counts.MissingModuleAttrs[path]++
					if clause.Internal != nil {
						imported[clause.Internal.Ident.Literal] = path
					} else {
						imported[clause.External.Ident.Literal] = path
					}
				}
			} else {
				for _, clause := range node.Names {
					if clause == nil || clause.External == nil {
						continue
					}
					if clauseNode, found := baseNode.Attr(clause.External.Ident.Literal); clauseNode == nil || !found {
						path := base + "." + clause.External.Ident.Literal
						counts.MissingModuleAttrs[path]++
						if clause.Internal != nil {
							imported[clause.Internal.Ident.Literal] = path
						} else {
							imported[clause.External.Ident.Literal] = path
						}
					}
				}
			}
			return false
		default:
			return true
		}
	})
	return imported
}

func (counts *counts) countAttrs(collector *collector, imported map[string]string, ast *pythonast.Module) {
	pythonast.InspectEdges(ast, func(parent, child pythonast.Node, field string) bool {
		if pythonast.IsNil(child) || pythonast.IsNil(parent) {
			return false
		}

		if _, isAssign := parent.(*pythonast.AssignStmt); isAssign && field == "Targets" {
			// do not recurse into LHS of assignment statements to avoid double counting
			return false
		}

		switch child := child.(type) {
		case *pythonast.ImportFromStmt, *pythonast.ImportNameStmt:
			// handle imports separately
			return false
		case *pythonast.NameExpr:
			if !collector.missingNames[child] {
				return false
			}
			if external := imported[child.Ident.Literal]; external != "" {
				if _, found := counts.MissingModules[external]; found {
					counts.MissingModules[external]++
				}
				if _, found := counts.MissingModuleAttrs[external]; found {
					counts.MissingModuleAttrs[external]++
				}
			}
			return false
		case *pythonast.AttributeExpr:
			if collector.missingAttrs[child] == nil {
				return false
			}
			if external, isExternal := collector.missingAttrs[child].Base.(pythontype.External); isExternal {
				if external.Node == nil {
					log.Fatalf("nil node for external for attr %s\n", pythonast.String(child))
				}

				if external.Node.Classification == pythonimports.Object && external.Node.Type != nil {
					base := external.Node.Type.CanonicalName.String()
					if counts.MissingClassAttrs[base] == nil {
						counts.MissingClassAttrs[base] = make(map[string]int64)
					}
					counts.MissingClassAttrs[base][child.Attribute.Literal]++
				} else {
					counts.MissingModuleAttrs[external.Node.CanonicalName.String()+"."+child.Attribute.Literal]++
				}
			}
			return false
		default:
			return true
		}
	})
}

func (counts *counts) count(graph *pythonimports.Graph, assembly *pythonstatic.Assembly, collecor *collector) {
	for _, file := range assembly.Files {
		imported := counts.countImports(graph, assembly.Sources, file)
		counts.countAttrs(collecor, imported, file.ASTBundle.AST)
	}
}

type missingAttr struct {
	Base pythontype.Value
}

// collector implements PropagatorDelegate; it recieves callbacks from the propagator
// and builds up the list of references and missing expressions
type collector struct {
	missingNames map[*pythonast.NameExpr]bool
	missingAttrs map[*pythonast.AttributeExpr]*missingAttr
}

// MissingName implements pythonstatic.PropagatorDelegate.
func (c *collector) MissingName(expr *pythonast.NameExpr) {
	// Cannot resolve this name expression. If it was being evaluated or deleted
	// then add it to the list of failures.
	if expr.Usage == pythonast.Evaluate || expr.Usage == pythonast.Delete {
		c.missingNames[expr] = true
	}
}

// MissingAttr implements pythonstatic.PropagatorDelegate.
func (c *collector) MissingAttr(expr *pythonast.AttributeExpr, base pythontype.Value) {
	// Cannot resolve this name expression. If it was being evaluated or deleted
	// then add it to the list of failures.
	if expr.Usage == pythonast.Evaluate || expr.Usage == pythonast.Delete {
		c.missingAttrs[expr] = &missingAttr{
			Base: base,
		}
	}
}

// MissingCall implements pythonstatic.PropagatorDelegate.
func (c *collector) MissingCall(expr *pythonast.CallExpr) {}

// Resolved implements pythonstatic.PropagatorDelegate
func (c *collector) Resolved(expr pythonast.Expr, val pythontype.Value) {}

type countArgs struct {
	CorpusDir string `arg:"positional"`
	Out       string `arg:"positional"`
	Graph     string
}

// Handle loads relevant data sets and walks user directories and runs data flow prop
func (args *countArgs) Handle() error {
	switch args.Graph {
	case "small":
		args.Graph = pythonimports.SmallImportGraph
	case "current":
		args.Graph = pythonimports.DefaultImportGraph
	case "":
		args.Graph = pythonimports.DefaultImportGraph
	}

	// global graph
	graph, err := pythonimports.NewGraph(args.Graph)
	if err != nil {
		return fmt.Errorf("error loading import graph: %v\n", err)
	}
	if err := pythonskeletons.UpdateGraph(graph); err != nil {
		return fmt.Errorf("error updating graph with skeletons: %v\n", err)
	}

	// type induction
	typeinducer, err := typeinduction.LoadModel(graph, typeinduction.DefaultClientOptions)
	if err != nil {
		return fmt.Errorf("error loading type induction client: %v\n", err)
	}
	pythonskeletons.UpdateReturnTypes(graph, typeinducer)

	counts := newCounts()
	dirs, err := ioutil.ReadDir(args.CorpusDir)
	if err != nil {
		return fmt.Errorf("error reading corpus dir `%s`: %v\n", args.CorpusDir, err)
	}
	for _, di := range dirs {
		if !di.IsDir() {
			continue
		}

		// walk corpus for user
		corpus := filepath.Join(args.CorpusDir, di.Name())
		var tooLarge, parseErrs, added, files int
		assembler := pythonstatic.NewAssembler(graph, typeinducer, pythonstatic.DefaultOptions)
		if err := filepath.Walk(corpus, func(path string, fi os.FileInfo, err error) error {
			switch {
			case err != nil:
				return err
			case !strings.HasSuffix(path, ".py"):
				return nil
			}
			files++

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("error reading file `%s`: %v", path, err)
			}

			if len(content) > largeFileThreshold {
				tooLarge++
				return nil
			}

			mod, err := pythonparser.Parse(kitectx.Background(), content, pythonparser.Options{
				ErrorMode: pythonparser.FailFast,
				ScanOptions: pythonscanner.Options{
					Label: path,
				},
			})

			if err != nil {
				parseErrs++
				return nil
			}

			abs, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("unable to make absolute path for `%s`: %v\n", path, err)
			}

			assembler.AddSource(pythonstatic.ASTBundle{AST: mod, Path: abs, Imports: pythonstatic.FindImports(mod)})
			added++
			return nil
		}); err != nil {
			return fmt.Errorf("failed to walk directory `%s`: %v\n", di.Name(), err)
		}

		collector := &collector{
			missingNames: make(map[*pythonast.NameExpr]bool),
			missingAttrs: make(map[*pythonast.AttributeExpr]*missingAttr),
		}
		assembly := assembler.Build(collector, false)

		counts.count(graph, assembly, collector)

		fmt.Printf("for %s:, of %d files: %d contained parse errors, %d were too large, added %d to batch\n",
			corpus, files, parseErrs, tooLarge, added)
	}

	if err := serialization.Encode(args.Out, counts); err != nil {
		return fmt.Errorf("error writing counts `%s`: %v\n", args.Out, err)
	}
	return nil
}
