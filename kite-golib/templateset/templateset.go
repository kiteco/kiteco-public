package templateset

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	texttemplate "text/template"
)

// Set encapsulates a set of templates
type Set struct {
	fs          http.FileSystem
	templateDir string
	funcMap     template.FuncMap
	ErrHandler  func(io.Writer, error)
}

// NewSet builds a new templateset.Set given a http.FileSystem and a directory.
func NewSet(fs http.FileSystem, dir string, funcMap template.FuncMap) *Set {
	return &Set{
		fs:          fs,
		templateDir: dir,
		funcMap:     funcMap,
		ErrHandler: func(w io.Writer, err error) {
			panic(err)
		},
	}
}

// RenderText renders the template as text with the given, passing along the provided
// payload, and writes the result to the given io.Writer.
func (s *Set) RenderText(w io.Writer, templateName string, payload interface{}) error {
	templatePath := path.Join(s.templateDir, templateName)

	file, err := s.fs.Open(templatePath)
	if err != nil {
		return fmt.Errorf("error opening template %s: %s", templatePath, err)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error reading template %s: %s", templatePath, err)
	}

	template, err := texttemplate.New(templateName).Parse(string(data))
	if err != nil {
		return fmt.Errorf("error parsing template %s: %s", templatePath, err)
	}

	err = template.Execute(w, payload)
	if err != nil {
		return fmt.Errorf("error executing template %s: %s", templatePath, err)
	}

	return nil
}

// Render renders the template as HTML with the given, passing along the provided
// payload, and writes the result to the given io.Writer.
func (s *Set) Render(w io.Writer, templateName string, payload interface{}) error {
	templatePath := path.Join(s.templateDir, templateName)

	file, err := s.fs.Open(templatePath)
	if err != nil {
		return fmt.Errorf("error opening template %s: %s", templatePath, err)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error reading template %s: %s", templatePath, err)
	}

	template, err := template.New(templateName).Funcs(s.funcMap).Parse(string(data))
	if err != nil {
		return fmt.Errorf("error parsing template %s: %s", templatePath, err)
	}

	err = template.Execute(w, payload)
	if err != nil {
		return fmt.Errorf("error executing template %s: %s", templatePath, err)
	}

	return nil
}

// MustRender renders a template like Render, but if it encounters an error it
// calls s.ErrHandler instead.
func (s *Set) MustRender(w io.Writer, templateName string, payload interface{}) {
	err := s.Render(w, templateName, payload)
	if err != nil {
		s.ErrHandler(w, err)
	}
}

// Validate checks whether there are any parse errors for the template files.
func (s *Set) Validate() error {
	dir, err := s.fs.Open(s.templateDir)
	if err != nil {
		return fmt.Errorf("error opening template dir %s: %v", s.templateDir, err)
	}
	entries, err := dir.Readdir(-1)
	if err != nil {
		return fmt.Errorf("error listing template dir %s: %v", s.templateDir, err)
	}
	for _, fileinfo := range entries {
		if !strings.HasSuffix(fileinfo.Name(), ".html") {
			continue
		}
		templatePath := path.Join(s.templateDir, fileinfo.Name())
		file, err := s.fs.Open(templatePath)
		if err != nil {
			return fmt.Errorf("cannot open template file %s: %v", templatePath, err)
		}
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return fmt.Errorf("cannot read the template file %s: %v", templatePath, err)
		}
		_, err = template.New(templatePath).Funcs(s.funcMap).Parse(string(data))
		if err != nil {
			return fmt.Errorf("error parsing template %s: %v", templatePath, err)
		}
	}
	return nil
}
