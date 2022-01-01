package templates

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
)

// ListFunc represents a function from directory paths to lists of file contents
type ListFunc func(string) ([]string, error)

// AssetFunc represents a function from paths to file contents
type AssetFunc func(string) ([]byte, error)

// TemplateSet represents a set of templates
type TemplateSet interface {
	// Render executes the named template to an output stream
	Render(w io.Writer, name string, payload interface{}) error
}

// PreloadedTemplateSet represents a set of templates loaded into memory
type PreloadedTemplateSet map[string]*template.Template

// NewTemplateSetFromBindata creates a template set using pointers to AssetDir and Asset
func NewTemplateSetFromBindata(AssetDir ListFunc, Asset AssetFunc, funcMap template.FuncMap) TemplateSet {
	templates := make(PreloadedTemplateSet)
	filenames, err := AssetDir("templates")
	if err != nil {
		log.Fatal(err)
	}
	for _, name := range filenames {
		log.Println("Loading template", name)
		data, err := Asset(path.Join("templates", name))
		if err != nil {
			log.Fatalf("error loading pre-compiled binary template data: %v\n", err)
			return nil
		}
		template, err := template.New(name).Funcs(funcMap).Parse(string(data))
		templates[name] = template
		if err != nil {
			log.Fatalf("error parsing template %s: %v\n", name, err)
			return nil
		}
	}
	return templates
}

// Render executes the named template to an output stream
func (p PreloadedTemplateSet) Render(w io.Writer, name string, payload interface{}) error {
	template, ok := p[name]
	if !ok {
		return fmt.Errorf("no template named %s", name)
	}

	err := template.Execute(w, payload)
	if err != nil {
		return fmt.Errorf("error rendering %s template: %v", name, err)
	}
	return nil
}

// FilesystemTemplateSet reloads templates from disk on every invokation
type FilesystemTemplateSet struct {
	path    string
	funcMap template.FuncMap
}

// NewTemplateSetFromDir creates a template set using pointers to AssetDir and Asset
func NewTemplateSetFromDir(path string, funcMap template.FuncMap) TemplateSet {
	return &FilesystemTemplateSet{path: path, funcMap: funcMap}
}

// Render executes the named template to an output stream
func (p *FilesystemTemplateSet) Render(w io.Writer, name string, payload interface{}) error {
	// Read file
	path := filepath.Join(p.path, name)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not load a template from %s: %v", path, err)
	}

	// Parse template
	template, err := template.New(name).Funcs(p.funcMap).Parse(string(data))
	if err != nil {
		return fmt.Errorf("error parsing template %s: %v", name, err)
	}

	// Execute template
	err = template.Execute(w, payload)
	if err != nil {
		return fmt.Errorf("error rendering template %s: %v", name, err)
	}
	return nil
}
