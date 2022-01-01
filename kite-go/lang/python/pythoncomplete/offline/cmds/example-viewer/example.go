package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/token"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/kitectx"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/example"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/scoring"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/highlight"
	"github.com/kiteco/kiteco/kite-go/response"
)

func (a *app) handleExample(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	var idx int
	idxParam := params.Get("idx")
	if idxParam != "" {
		idx64, err := strconv.ParseInt(idxParam, 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("error parsing 'idx' param: %v", err), http.StatusBadRequest)
			return
		}
		idx = int(idx64)
	}

	if idx < 0 {
		idx = len(a.collection.Examples) - 1
	} else if idx >= len(a.collection.Examples) {
		idx = 0
	}

	ex := a.collection.Examples[int(idx)]
	if ex.Cursor == 0 {
		src := ex.Buffer
		idx := strings.Index(src, "$$$")
		if idx > 0 {
			ex.Cursor = int64(idx)
			ex.Buffer = src[:ex.Cursor] + src[ex.Cursor+3:]
		}
	}

	buf := ex.Buffer
	bufProvider := params.Get("buffer")
	if bufProvider != "" {
		p, ok := ex.Provided[bufProvider]
		if !ok {
			http.Error(w, fmt.Sprintf("bad buffer param: could not find '%s' provider", bufProvider),
				http.StatusBadRequest)
			return
		}
		if p.MungedBuffer != "" {
			buf = p.MungedBuffer
		}
	}

	viewMode := params.Get("view")
	var viewHTML string
	var err error
	if viewMode == "ast" {
		viewHTML, err = a.renderAST(idx, buf, ex)
		if err != nil {
			http.Error(w, fmt.Sprintf("error rendering AST: %v", err.Error()), http.StatusInternalServerError)
			return
		}
	} else {
		viewHTML, err = a.renderCompletions(idx, ex)
		if err != nil {
			http.Error(w, fmt.Sprintf("error rendering completions: %v", err.Error()), http.StatusInternalServerError)
			return
		}
	}

	code, err := highlight.Highlight(buf, ex.Cursor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	exURL := func(idx int) string {
		if idx < 0 {
			idx = len(a.collection.Examples) - 1
		} else if idx >= len(a.collection.Examples) {
			idx = 0
		}
		url := fmt.Sprintf("/example?idx=%d", idx)
		if bufProvider != "" {
			url += "&buffer=" + bufProvider
		}
		return url
	}

	playgroundURL := fmt.Sprintf("%s/?buffer=%s&cursor=%d", a.playgroundEndpoint,
		url.QueryEscape(ex.Buffer), ex.Cursor)

	err = a.templates.Render(w, "example.html", map[string]interface{}{
		"Example":       ex,
		"Code":          template.HTML(code),
		"RawBuffer":     ex.Buffer,
		"Idx":           idx,
		"Count":         len(a.collection.Examples),
		"View":          template.HTML(viewHTML),
		"PrevURL":       exURL(idx - 1),
		"NextURL":       exURL(idx + 1),
		"PlaygroundURL": playgroundURL,
		"ViewMode":      viewMode,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type comp struct {
	Comp        example.Completion
	Source      response.EditorCompletionSource
	Metadata    template.HTML
	Correct     bool
	Score       scoring.CompletionScore
	ScoreDetail template.HTML
	Duplicate   bool
}

func (a *app) renderCompletions(idx int, ex example.Example) (string, error) {
	type compType struct {
		Name string
		URL  string
	}

	/*
		compTypes := make([]compType, 0, len(ex.Provided))
		for k, c := range ex.Provided {
			compTypes = append(compTypes, compType{
				Name: k,
				URL:  fmt.Sprintf("/example?idx=%d&buffer=%s", idx, k),
			})
			if len(c.Completions) > maxComps {
				maxComps = len(c.Completions)
			}
		}
		sort.Slice(compTypes, func(i, j int) bool {
			return compTypes[i].Name < compTypes[j].Name
		})
	*/

	completions := a.completionProvider.GetCompletions(ex)
	comps := make([]comp, 0, len(completions))
	for _, c := range completions {
		score, _ := scoring.ScoreCompletion(c, ex.Expected)
		comps = append(comps, comp{
			Comp:        c,
			Source:      c.MixCompletion.MetaCompletion().Source,
			Metadata:    formatMetadata(c.MixCompletion.MetaCompletion()),
			Correct:     c.Identifier == ex.Expected,
			Score:       score,
			ScoreDetail: template.HTML(scoring.ScoreDetailString(score)),
			Duplicate:   c.Duplicate,
		})

	}
	selectedComps := selectCompletions(comps)

	var buf bytes.Buffer
	err := a.templates.Render(&buf, "completions.html", map[string]interface{}{
		"Comps":         comps,
		"SelectedComps": selectedComps,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func selectCompletions(comps []comp) []comp {
	var result []comp
	result = append(result, selectCompFromProvider(comps, response.GlobalPopularPatternCompletionSource, 2)...)
	result = append(result, selectCompFromProvider(comps, response.LocalPopularPatternCompletionSource, 2)...)
	result = append(result, selectCompFromProvider(comps, response.CallModelCompletionSource, 2)...)
	result = append(result, selectCompFromProvider(comps, response.ArgSpecCompletionSource, 1)...)
	return result
}

func selectCompFromProvider(comps []comp, provider response.EditorCompletionSource, count int) []comp {
	var result []comp
	for i := 0; i < len(comps) && count > 0; i++ {
		if comps[i].Source == provider && !comps[i].Duplicate {
			result = append(result, comps[i])
			count--
			if count == 0 {
				break
			}
		}

	}
	return result
}

func formatMetadata(meta pythonproviders.MetaCompletion) template.HTML {
	htmlTemplate := `
<div>
Model: %s
<div>
Raw json: %s
</div>
</div>`
	rawJSON, _ := json.Marshal(meta)
	return template.HTML(fmt.Sprintf(htmlTemplate, meta.Source, rawJSON))
}

func (a *app) renderAST(idx int, buf string, ex example.Example) (string, error) {
	cursor := token.Pos(ex.Cursor)
	ast, _ := pythonparser.Parse(kitectx.Background(), []byte(buf), pythonparser.Options{
		Approximate: true,
		Cursor:      &cursor,
		ErrorMode:   pythonparser.Recover,
	})

	var astBuf bytes.Buffer
	pythonast.Print(ast, &astBuf, "    ")

	var providers []string
	for p := range ex.Provided {
		providers = append(providers, p)
	}
	sort.Strings(providers)

	var rendered bytes.Buffer
	err := a.templates.Render(&rendered, "ast.html", map[string]interface{}{
		"Idx":       idx,
		"Providers": providers,
		"AST":       astBuf.String(),
	})
	if err != nil {
		return "", err
	}
	return rendered.String(), nil
}
