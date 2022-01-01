package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
)

// Combines the provided template paths by invoking their in-memory contents using Asset(path)
// and building up a template object incrementally using the tmpl.Parse function. The returned
// template is ready to be rendered.
func buildTemplate(tmplName string, assetNames ...string) (*template.Template, error) {
	return buildTemplateWithFuncs(tmplName, nil, assetNames...)
}

// As above, but also attaches the given function map to the template.
func buildTemplateWithFuncs(tmplName string, funcs template.FuncMap, assetNames ...string) (*template.Template, error) {
	tmpl := template.New(tmplName).Funcs(funcs)
	for _, assetName := range assetNames {
		asset, err := Asset(path.Join("templates", assetName))
		if err != nil {
			return nil, fmt.Errorf("Could not find template %s: %v", assetName, err)
		}
		_, err = tmpl.Parse(string(asset))
		if err != nil {
			return nil, fmt.Errorf("Error parsing template %s: %v", assetName, err)
		}
	}
	return tmpl, nil
}

// Build a tempalte from assets and execute it with the given data, sending output to the specified writer
func runTemplateWithFuncs(w io.Writer, data interface{}, funcs template.FuncMap, assetNames ...string) error {
	// Parse the template
	tmpl, err := buildTemplateWithFuncs("", funcs, assetNames...)
	if err != nil {
		return err
	}

	// Execute the templates
	return tmpl.Execute(w, data)
}

func runTemplate(w io.Writer, data interface{}, assetNames ...string) error {
	return runTemplateWithFuncs(w, data, nil, assetNames...)
}

func reportError(w http.ResponseWriter, s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	http.Error(w, s, http.StatusInternalServerError)
	log.Println(s)
}
