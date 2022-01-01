package pythondocs

import (
	"io"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	base = "PyQt4"
)

// ParsePyQt4HTML takes a PyQt4 html document and returns a Module object
// containing the language entities detected in the html document.
func ParsePyQt4HTML(r io.Reader, path string, coverage bool) *Module {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Fatal(err)
	}

	header := doc.Find("h1")
	if header == nil {
		log.Fatalf("Couldn't find header")
	}
	if header.Length() != 1 {
		log.Fatalf("Expect exactly one <h1>, got %d", header.Length())
	}

	// Is a module page
	var module *Module
	if split := strings.Split(header.Text(), " "); len(split) == 2 && split[len(split)-1] == "Module" {
		module = parseModule(base, split[0], doc)
	}

	// Is a class page
	if sup := header.Find("sup"); sup.Length() != 0 && module == nil {
		if anchor := sup.Find("a"); anchor.Text() != "" {
			module = parseClassReference(base, anchor.Text(), doc)
		}
	}

	if coverage {
		split := strings.Split(filepath.Base(path), ".")
		name := split[:len(split)-1][0]
		renderCoverage(doc, name)
	}

	return module
}

func parseClassReference(moduleName, subModule string, doc *goquery.Document) *Module {
	module := &Module{
		Name: moduleName,
	}
	extractPyQt4Class(module, subModule, doc)
	extractPyQt4ClassMethods(module, subModule, doc)
	extractPyQt4ClassAttributes(module, subModule, doc)
	// TODO: extract description of module as well
	return module
}

func parseModule(moduleName, subModule string, doc *goquery.Document) *Module {
	module := &Module{
		Name: moduleName,
	}
	extractPyQt4ModuleFunctions(module, subModule, doc)
	extractPyQt4ModuleVariables(module, subModule, doc)
	// TODO: extract description of module as well
	return module
}

func extractPyQt4Class(module *Module, subModule string, doc *goquery.Document) {
	header := doc.Find("h1").First()
	if header == nil {
		return
	}

	className := extractClassName(doc)

	details := doc.Find("a[name=details]")
	if details.Length() == 0 {
		return
	}
	var description string
	details.Children().AddClass("kite_highlight")
	details.NextUntil("hr").AddClass("kite_highlight").Each(func(j int, g *goquery.Selection) {
		h, err := g.Html()
		if err != nil {
			log.Printf("Error parsing description of %s: %v", g.Text(), err)
		}
		description += h
	})

	module.Classes = append(module.Classes, &LangEntity{
		Module:    module.Name,
		Kind:      ClassKind,
		Ident:     module.Name + "." + subModule,
		Sel:       className,
		Signature: "",
		Doc:       cleanHTML(description),
		DocHTML:   description,
	})
}

func extractPyQt4ClassMethods(module *Module, subModule string, doc *goquery.Document) {
	subHeaders := doc.Find("h3")
	if subHeaders == nil {
		return
	}

	className := extractClassName(doc)
	methodDescriptions := extractDescriptions(doc, "Method Documentation")
	for k, v := range extractDescriptions(doc, "Qt Signal Documentation") {
		methodDescriptions[k] = v
	}

	subHeaders.Each(func(i int, s *goquery.Selection) {
		if !strings.Contains(s.Text(), "Methods") && !strings.Contains(s.Text(), "Qt Signals") {
			return
		}
		methodsList := s.Next()
		if !methodsList.Is("ul") {
			return
		}
		methodsList.Find("li").Each(func(j int, g *goquery.Selection) {
			h, err := g.Html()
			if err != nil {
				log.Printf("Error parsing description of %s: %v", g.Text(), err)
			}
			sig := strings.TrimSpace(h)
			module.ClassMethods = append(module.ClassMethods, &LangEntity{
				Module:        module.Name,
				Kind:          MethodKind,
				Ident:         module.Name + "." + subModule + "." + className,
				Sel:           g.Find("b a").First().Text(),
				Signature:     cleanHTML(sig),
				SignatureHTML: sig,
				Doc:           cleanHTML(methodDescriptions[g.Text()]),
				DocHTML:       methodDescriptions[g.Text()],
			})
			g.AddClass("kite_highlight")
		})
	})
}

func extractPyQt4ClassAttributes(module *Module, subModule string, doc *goquery.Document) {
	subHeaders := doc.Find("h3")
	if subHeaders == nil {
		return
	}

	className := extractClassName(doc)
	typeDescriptions := extractDescriptions(doc, "Type Documentation")

	subHeaders.Each(func(i int, s *goquery.Selection) {
		if !strings.Contains(s.Text(), "Types") {
			return
		}
		typesList := s.Next()
		if !typesList.Is("ul") {
			return
		}
		typesList.Find("li").Each(func(j int, g *goquery.Selection) {
			h, err := g.Html()
			if err != nil {
				log.Printf("Error parsing description of %s: %v", g.Text(), err)
			}
			sig := strings.TrimSpace(h)
			module.ClassAttributes = append(module.ClassAttributes, &LangEntity{
				Module:        module.Name,
				Kind:          AttributeKind,
				Ident:         module.Name + "." + subModule + "." + className,
				Sel:           g.Find("b a").First().Text(),
				Signature:     cleanHTML(sig),
				SignatureHTML: sig,
				Doc:           cleanHTML(typeDescriptions[g.Text()]),
				DocHTML:       typeDescriptions[g.Text()],
			})
			g.AddClass("kite_highlight")
		})
	})
}

func extractPyQt4ModuleFunctions(module *Module, subModule string, doc *goquery.Document) {
	subHeaders := doc.Find("h3")
	if subHeaders == nil {
		return
	}

	functionDescriptions := extractDescriptions(doc, "Function Documentation")

	subHeaders.Each(func(i int, s *goquery.Selection) {
		if !strings.Contains(s.Text(), "Module Functions") {
			return
		}
		functionsList := s.Next()
		if !functionsList.Is("ul") {
			return
		}
		functionsList.Find("li").Each(func(j int, g *goquery.Selection) {
			h, err := g.Html()
			if err != nil {
				log.Printf("Error parsing description of %s: %v", g.Text(), err)
			}
			sig := strings.TrimSpace(h)
			module.Funcs = append(module.Funcs, &LangEntity{
				Module:        module.Name,
				Kind:          FunctionKind,
				Ident:         module.Name + "." + subModule,
				Sel:           g.Find("b a").First().Text(),
				Signature:     cleanHTML(sig),
				SignatureHTML: sig,
				Doc:           cleanHTML(functionDescriptions[g.Text()]),
				DocHTML:       functionDescriptions[g.Text()],
			})
			g.AddClass("kite_highlight")
		})
	})
}

func extractPyQt4ModuleVariables(module *Module, subModule string, doc *goquery.Document) {
	subHeaders := doc.Find("h3")
	if subHeaders == nil {
		return
	}

	variableDescriptions := extractDescriptions(doc, "Member Documentation")

	subHeaders.Each(func(i int, s *goquery.Selection) {
		if !strings.Contains(s.Text(), "Module Members") {
			return
		}
		variablesList := s.Next()
		if !variablesList.Is("ul") {
			return
		}
		variablesList.Find("li").Each(func(j int, g *goquery.Selection) {
			h, err := g.Html()
			if err != nil {
				log.Printf("Error parsing description of %s: %v", g.Text(), err)
			}
			sig := strings.TrimSpace(h)
			module.Vars = append(module.Vars, &LangEntity{
				Module:        module.Name,
				Kind:          VariableKind,
				Ident:         module.Name + "." + subModule,
				Sel:           g.Find("b a").First().Text(),
				Signature:     cleanHTML(sig),
				SignatureHTML: sig,
				Doc:           cleanHTML(variableDescriptions[g.Text()]),
				DocHTML:       variableDescriptions[g.Text()],
			})
			g.AddClass("kite_highlight")
		})
	})
}

// helpers

func extractClassName(doc *goquery.Document) string {
	header := doc.Find("h1").First()
	if header == nil {
		return ""
	}
	pattern, err := regexp.Compile(`([a-zA-Z0-9]+)\sClass\sReference`)
	if err != nil {
		log.Printf("Error compiling regexp: %v\n", err)
		return ""
	}
	return pattern.FindStringSubmatch(header.Text())[1]
}

// Build a map of method/class/type full signature --> it's description, if any.
func extractDescriptions(doc *goquery.Document, header string) map[string]string {
	subHeaders := doc.Find("h2")
	if subHeaders == nil {
		return nil
	}

	descriptions := make(map[string]string)
	subHeaders.Each(func(i int, s *goquery.Selection) {
		if !strings.Contains(s.Text(), header) {
			return
		}
		s.NextUntil("hr").Each(func(j int, g *goquery.Selection) {
			if !g.Is("h3") {
				return
			}
			var description string
			g.NextUntil("h3").AddClass("kite_highlight").Each(func(j int, k *goquery.Selection) {
				h, err := k.Html()
				if err != nil {
					log.Printf("Error parsing description of %s: %v", k.Text(), err)
				}
				description += h
			})
			descriptions[g.Text()] = description
			g.AddClass("kite_highlight")
		})
		s.AddClass("kite_highlight")
	})
	return descriptions
}
