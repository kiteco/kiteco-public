package main

import (
	"go/scanner"
	"go/token"
	"log"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/github"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
)

type replacer struct {
	pattern *regexp.Regexp
	repl    string
}

// an exhaustive list of rules for text normalization
var (
	textNormalizer = []replacer{
		replacer{pattern: regexp.MustCompile(` \. `), repl: `.`},
		replacer{pattern: regexp.MustCompile(` *\( *`), repl: `(`},
		replacer{pattern: regexp.MustCompile(` *\) *`), repl: `)`},
		replacer{pattern: regexp.MustCompile(` *\[ *`), repl: `[`},
		replacer{pattern: regexp.MustCompile(` *\] *`), repl: `]`},
		replacer{pattern: regexp.MustCompile("r *<STR>"), repl: `<STR>`},
		replacer{pattern: regexp.MustCompile(` , `), repl: `, `},
		replacer{pattern: regexp.MustCompile(";$"), repl: ``},
	}
)

func computeStats(packages []*github.Package) {
	sum := 0.0
	for _, p := range packages {
		sum += float64(p.Counts)
	}

	pCdf := 0.0
	for _, p := range packages {
		pFreq := float64(p.Counts) / sum
		pCdf += pFreq
		p.Cdf = pCdf
		p.Freq = pFreq

		sCdf := 0.0
		for _, b := range p.Submodules {
			sFreq := float64(b.Counts) / float64(p.Counts)
			sCdf += sFreq
			b.Cdf = sCdf
			b.Freq = sFreq

			mCdf := 0.0
			for _, m := range b.Methods {
				mFreq := float64(m.Count) / float64(b.Counts)
				mCdf += mFreq
				m.Cdf = mCdf
				m.Freq = mFreq
			}
		}
	}
}

func normalizeText(text string) string {
	buf := []byte(text)
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(buf))
	var lexer scanner.Scanner
	lexer.Init(file, buf, nil, 0)
	var words []string
	for {
		_, t, lit := lexer.Scan()
		if t == token.EOF {
			break
		}
		parsedWords := tokenToWords(t, lit)
		parsedWords = normalizeStringTypes(parsedWords)
		words = append(words, parsedWords...)
	}
	code := strings.Join(words, " ")
	for _, norm := range textNormalizer {
		code = norm.pattern.ReplaceAllString(code, norm.repl)
	}
	return code
}

func tokenToWords(t token.Token, lit string) []string {
	switch t {
	case token.ILLEGAL, token.EOF:
		// Do not generate any words for these tokens
		return nil
	case token.COMMENT, token.STRING:
		// If it's python, then we only care about the type.
		return []string{"<STR>"}
	case token.IDENT:
		return []string{lit}
	default:
		return []string{t.String()}
	}
}

// combine "CHAR" type and "STRING" type in python
func normalizeStringTypes(tokens []string) []string {
	for i, t := range tokens {
		if t == "CHAR" {
			tokens[i] = "<STR>"
		}
	}
	return tokens
}

func mostConcise(gs []*curation.GithubSnippet) int {
	minLen := len(gs[0].Statement)
	minIndex := 0
	for i := 0; i < len(gs); i++ {
		if len(gs[i].Statement) < minLen {
			minIndex = i
			minLen = len(gs[i].Statement)
		}
	}
	return minIndex
}

// loadTemplates loads the binary html templates compiled by go-bindata
func loadTemplates() {
	for _, name := range htmlTemplates {
		data, err := Asset(name)
		if err != nil {
			log.Fatal("Can't load pre-compiled binary template data.")
			return
		}
		template, err := template.New(name).Parse(string(data))
		templates[name] = template
		if err != nil {
			log.Fatal("Can't create a template for ", name)
			return
		}
	}
}

func renderTemplate(w http.ResponseWriter, name string, res interface{}) {
	if template, ok := templates[name]; !ok {
		webutils.ReportError(w, "can't load the template named %s", name)
	} else {
		err := template.Execute(w, res)
		if err != nil {
			webutils.ReportError(w, err.Error())
		}
	}
}

func findSubmodules(packageName string) []*github.Submodule {
	for _, p := range packages {
		if p.Name == packageName {
			return p.Submodules
		}
	}
	return nil
}
