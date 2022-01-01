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
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/argspec"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/pkg/errors"
)

func createSpec(def *pythonast.FunctionDefStmt, src []byte) (*argspec.Entity, error) {
	var args []pythonimports.Arg
	for _, p := range def.Parameters {
		if n, ok := p.Name.(*pythonast.NameExpr); ok {
			var defaultValue string
			if p.Default != nil {
				defaultValue = string(src[p.Default.Begin():p.Default.End()])
				if defaultValue == "" {
					// there is a default value, but it's empty for some reason; reset to "..."
					defaultValue = "..."
				}
			}

			arg := pythonimports.Arg{
				// explicitly omit DefaultType, Types to avoid complexity; TODO fix/extend
				Name:         n.Ident.Literal,
				DefaultValue: defaultValue,
				KeywordOnly:  p.KeywordOnly,
			}
			args = append(args, arg)
		} else {
			// ignore functions using tuple parameter unpacking syntax (Py2)
			// PEP-3113 removes it from Py3
			return nil, errors.Errorf("%d:%d:function parameters use tuple unpacking syntax", p.Begin(), p.End())
		}
	}

	var kwarg, vararg string
	if def.Vararg != nil {
		vararg = def.Vararg.Name.Ident.Literal
	}
	if def.Kwarg != nil {
		kwarg = def.Kwarg.Name.Ident.Literal
	}

	return &argspec.Entity{
		Args:   args,
		Vararg: vararg,
		Kwarg:  kwarg,
	}, nil
}

// TODO(naman) the right way to do this involves improving our internal repr of argspecs,
// and storing multiple argspecs per function
func unionSpecs(specs ...argspec.Entity) (*argspec.Entity, error) {
	var args []pythonimports.Arg

	// process args in positional order, rather than the order of input spec
	for i := 0; ; i++ {
		noArgs := true // if we find args to process, we'll set this to false

		var (
			hasDefaultValue bool

			keywordOnly  bool
			defaultValue string
			name         string
		)

		for _, spec := range specs {
			if len(spec.Args) > i {
				noArgs = false
				arg := spec.Args[i]

				// any spec with a *varargs before the parameter should make the parameter keyword only
				keywordOnly = keywordOnly || arg.KeywordOnly

				if name != "" && name != arg.Name {
					// if we find differently named parameters in the same position, we must fail
					return nil, errors.Errorf("argspec parameter name mismatch (%s vs %s)", name, arg.Name)
				}
				name = arg.Name

				if arg.DefaultValue != "" {
					switch defaultValue {
					case arg.DefaultValue:
						// they're the same; nothing to do
					case "":
						defaultValue = arg.DefaultValue
					default:
						// if we find different default values for parameters in the same position, warn and fall back to "..."
						log.Printf("[WARN] argspec default value mismatch (%s vs %s); falling back to ...\n", defaultValue, arg.DefaultValue)
						defaultValue = "..."
						continue
					}
				}
			} else {
				// don't just set defaultValue = "..." here, because we might find a non-ellipsis default value in a subsequent iteration
				hasDefaultValue = true
			}
		}

		if noArgs {
			break
		}

		// if we didn't find a non-... default value, but there must be one, set it to "..."
		if hasDefaultValue && defaultValue == "" {
			defaultValue = "..."
		}

		args = append(args, pythonimports.Arg{
			Name:         name,
			DefaultValue: defaultValue,
			KeywordOnly:  keywordOnly,
		})
	}

	var vararg, kwarg string
	for _, spec := range specs {
		if spec.Vararg != "" {
			if vararg != "" && vararg != spec.Vararg {
				// if the vararg names aren't all the same, warn and choose the lexicographically smallest
				log.Printf("[WARN] argspec vararg name mismatch (%s vs %s)\n", vararg, spec.Vararg)
				if vararg < spec.Vararg {
					vararg = spec.Vararg
				}
			} else {
				vararg = spec.Vararg
			}
		}

		if spec.Kwarg != "" {
			// if the kwarg names aren't all the same, warn and choose the lexicographically smallest
			if kwarg != "" && kwarg != spec.Kwarg {
				log.Printf("[WARN] argspec vararg name mismatch (%s vs %s)\n", kwarg, spec.Kwarg)
				if kwarg < spec.Kwarg {
					kwarg = spec.Kwarg
				}
			} else {
				kwarg = spec.Kwarg
			}
		}
	}

	return &argspec.Entity{
		Args:   args,
		Vararg: vararg,
		Kwarg:  kwarg,
	}, nil
}

type collector map[string][]argspec.Entity

// stmt recursively collects defined argspecs from a function/class definition statement
func (c collector) stmt(prefix string, stmt pythonast.Stmt, src []byte) {
	switch stmt := stmt.(type) {
	case *pythonast.IfStmt:
		for _, branch := range stmt.Branches {
			for _, subStmt := range branch.Body {
				c.stmt(prefix, subStmt, src)
			}
		}
		for _, subStmt := range stmt.Else {
			c.stmt(prefix, subStmt, src)
		}
	case *pythonast.FunctionDefStmt:
		functionName := stmt.Name.Ident.Literal

		var fullname string
		var secondaryName string
		switch true {
		case functionName == "__init__":
			fullname = prefix // use the parent (class) name
			secondaryName = fmt.Sprintf("%s.%s", prefix, functionName)
			// We register the init argSpec on both the type name and the __init__ function
		case strings.HasPrefix(functionName, "__"):
			// ignore non-`__init__` dunder methods for the purposes of argspecs
			return
		default:
			fullname = fmt.Sprintf("%s.%s", prefix, functionName)
		}

		spec, err := createSpec(stmt, src)
		if err != nil || spec == nil {
			log.Printf("[ERROR] cannot generate argspec for function %s: %s", fullname, err)
			return
		}

		c[fullname] = append(c[fullname], *spec)
		if secondaryName != "" {
			c[secondaryName] = append(c[secondaryName], *spec)
		}
	case *pythonast.ClassDefStmt:
		className := stmt.Name.Ident.Literal
		fqClassName := fmt.Sprintf("%s.%s", prefix, className)

		for _, subStmt := range stmt.Body {
			c.stmt(fqClassName, subStmt, src)
		}

		// TODO(naman) if we didn't recursively find an __init__ for the class argspec, then we should try to
		// 1. search superclasses, and 2. fall back to default `def __init__(): ...`
	}
	return
}

// pyi traverses the AST of a pyi file to collect all defined function & type argspecs
// using the provided module path as a namespace/prefix for the symbol names
func (c collector) pyi(pyiPath string, modulePath string) error {
	src, err := ioutil.ReadFile(pyiPath)
	if err != nil {
		return errors.Wrapf(err, "error reading pyi file %s", pyiPath)
	}

	ast, err := pythonparser.Parse(kitectx.Background(), src, pythonparser.Options{})
	if err != nil {
		return errors.Wrapf(err, "error parsing pyi file %s", pyiPath)
	}

	for _, stmt := range ast.Body {
		c.stmt(modulePath, stmt, src)
	}
	return nil
}

// pathMap computes a map from module paths to source file paths
func pathMap(packageRoot string) map[string]string {
	sourceMap := make(map[string]string)

	filepath.Walk(packageRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalln("[FATAL] error walking pyi package", packageRoot, err)
		}
		if info.IsDir() || filepath.Ext(path) != ".pyi" {
			return nil
		}

		relpath, err := filepath.Rel(packageRoot, path)
		if err != nil {
			log.Fatalln("[FATAL] filepath.Rel error", err)
		}

		// compute the module path, almost by replacing / with . in the file path
		components := strings.Split(filepath.ToSlash(relpath), "/")
		if filename := components[len(components)-1]; filename == "__init__.pyi" {
			// __init__.pyi modules should inherit the module path of the containing package
			components = components[:len(components)-1]
		} else {
			components[len(components)-1] = strings.TrimSuffix(filename, ".pyi")
		}
		modulePath := strings.Join(components, ".")

		sourceMap[modulePath] = path
		return nil
	})
	return sourceMap
}

// collect computes all defined function/class argspecs inside a pyi package
// and additionally returns a slice of (ignored) errors
func collect(packageRoot string) map[string]argspec.Entity {
	stubs := pathMap(packageRoot)

	c := make(collector)
	for module, pyi := range stubs {
		err := c.pyi(pyi, module)
		if err != nil { // explicitly log and ignore parsing errors
			log.Println("[ERROR]", err)
			continue
		}
	}

	out := make(map[string]argspec.Entity)
	for name, specs := range c {
		merged, err := unionSpecs(specs...)
		if err != nil || merged == nil {
			log.Printf("[ERROR] ommiting function %s; failed to merge overloaded argspecs: %s", name, err)
			continue
		}
		out[name] = *merged
	}

	return out
}
