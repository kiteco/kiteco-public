package main

import (
	"compress/gzip"
	"encoding/json"
	"log"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonranker"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// loadPythonModeuls loads python module documentation
func loadPythonModeuls(path string) pythondocs.Modules {
	modules := make(pythondocs.Modules)
	file, err := fileutil.NewCachedReader(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	decomp, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}

	dec := json.NewDecoder(decomp)
	err = modules.Decode(dec)
	if err != nil {
		log.Fatal(err)
	}
	return modules
}

// loadDocs load the given docs corpus.
// Right now, we only do classification on class methods and
// global functions, which we may want to change in the future.
func loadDocs(path string) map[string]map[string]*pythonranker.MethodTrainingData {
	modules := loadPythonModeuls(path)
	packageToData := make(map[string]map[string]*pythonranker.MethodTrainingData)
	for n, module := range modules {
		data := make(map[string]*pythonranker.MethodTrainingData)
		categories := [][]*pythondocs.LangEntity{
			module.ClassMethods,
			module.Funcs,
		}
		// go through docs for the categories that we care.
		for _, cat := range categories {
			for _, d := range cat {
				var md *pythonranker.MethodTrainingData
				var exists bool
				if md, exists = data[d.Sel]; !exists {
					md = &pythonranker.MethodTrainingData{}
					data[d.Sel] = md
				}
				md.Data = append(md.Data, processor.Apply(tokenizer(d.Doc))...)
			}
		}
		packageToData[n] = data
	}
	return packageToData
}

// parseDocStruct goes through the doc corpus, and construct
// a hierarchical representation of each package.
func parseDocStruct(path string) ([]*parsedModule, map[string][]string) {
	modules := loadPythonModeuls(path)

	var parsedModules []*parsedModule
	matchingCandidates := make(map[string][]string)

	for p, module := range modules {
		seenNames := make(map[string]struct{})
		parsedModule := &parsedModule{
			name: p,
		}
		classMap := make(map[string]*class)
		for _, c := range module.Classes {
			name := strings.Join([]string{c.Ident, c.Sel}, ".")
			newClass := &class{
				name: name,
			}
			parsedModule.classes = append(parsedModule.classes, newClass)
			classMap[name] = newClass
			seenNames[c.Sel] = struct{}{}
		}
		for _, m := range module.ClassMethods {
			if class, exists := classMap[m.Ident]; exists {
				class.methods = append(class.methods, &method{name: m.Sel})
			} else {
				parsedModule.functions = append(parsedModule.functions, &function{name: m.Sel})
			}
			seenNames[m.Sel] = struct{}{}
		}
		for _, f := range module.Funcs {
			name := strings.Join([]string{f.Ident, f.Sel}, ".")
			parsedModule.functions = append(parsedModule.functions, &function{name: name})
			seenNames[f.Sel] = struct{}{}
		}
		parsedModules = append(parsedModules, parsedModule)
		var candidates []string
		for c := range seenNames {
			candidates = append(candidates, c)
		}
		matchingCandidates[p] = candidates
	}
	return parsedModules, matchingCandidates
}

// parsedModule is a module with structured internal representation.
// It contains its classes and its global functions. Each of the classes
// contains its member methods.
type parsedModule struct {
	name      string
	classes   []*class
	functions []*function
}

// estimateCount estimates the count of the selector name.
// If the selector matches with any package global functions,
// then the raw caount of the global function is returned.
// Otherwise, we check whether the selector name matches with
// any class's memeber function, and estimate the count for
// selector name.
func (pm parsedModule) estimateCount(sel string) float64 {
	if function := pm.findFunction(sel); function != nil {
		return float64(function.count)
	}
	var total float64
	for _, c := range pm.classes {
		total += c.estimateCount(sel)
	}
	return total
}

// findEntity finds the entiry that has the given selector name.
func (pm parsedModule) findEntity(s string) entity {
	if function := pm.findFunction(s); function != nil {
		return function
	}
	if class := pm.findClass(s); class != nil {
		return class
	}
	tokens := strings.Split(s, ".")
	if len(tokens) > 1 {
		ident := strings.Join(tokens[:len(tokens)-1], ".")
		sel := tokens[len(tokens)-1]
		if class := pm.findClass(ident); class != nil {
			if method := class.findClassMethod(sel); method != nil {
				return method
			}
		}
	}
	return nil
}

// findClassMethod returns the the method entity that has
// the given name as its name suffix.
func (c *class) findClassMethod(name string) *method {
	for _, m := range c.methods {
		if strings.HasSuffix(m.name, name) {
			return m
		}
	}
	return nil
}

// estimateCount estimates the count of the selector name by using
// the rough class method distribution we get from github, and multiple
// that with the number of observations for the class contructor.
// Note that this estimation probably is quite off.
func (c *class) estimateCount(sel string) float64 {
	var ratio float64
	if m := c.findClassMethod(sel); m != nil {
		var maxCount int
		for _, method := range c.methods {
			if method.count > maxCount {
				maxCount = method.count
			}
		}
		ratio = float64(m.count+1) / float64(maxCount+1)
	}
	return float64(c.count) * ratio
}

// findClass returns the class entity that has the name as the given
// selector name.
func (pm parsedModule) findClass(name string) *class {
	for _, c := range pm.classes {
		if strings.HasSuffix(c.name, name) {
			return c
		}
	}
	return nil
}

// findFunction returns the function entity that has the name the same
// as the given selector name.
func (pm parsedModule) findFunction(name string) *function {
	for _, c := range pm.functions {
		if strings.HasSuffix(c.name, name) {
			return c
		}
	}
	return nil
}

type parsedModules []*parsedModule

// find returns the parsed module object whose name is the same as the given name.
func (pms parsedModules) find(p string) *parsedModule {
	for _, pm := range pms {
		if pm.name == p {
			return pm
		}
	}
	return nil
}

// an entity can be a class object or a method object, or a function object.
type entity interface {
	addCount(int)
	ident() string
}

// class represents a class in a module, which contains methods.
type class struct {
	name    string
	methods []*method
	count   int
}

func (c *class) addCount(n int) {
	c.count += n
}

func (c *class) ident() string {
	return c.name
}

// function represents a global function in a module.
type function struct {
	name  string
	count int
}

func (f *function) addCount(n int) {
	f.count += n
}

func (f *function) ident() string {
	return f.name
}

// method represents a class member function.
type method struct {
	name  string
	count int
}

func (m *method) addCount(n int) {
	m.count += n
}

func (m *method) ident() string {
	return m.name
}
