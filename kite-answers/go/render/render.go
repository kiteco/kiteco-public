package render

import (
	"strings"
	"unicode"

	"github.com/kiteco/kiteco/kite-answers/go/execution"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// OutputBlock encapsulates sandbox-computed output
type OutputBlock struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Data  string `json:"data"`
}

// CodeBlockItem is either a line of code, or an output block
// Each CodeBlockItem may have an optional annotation block
type CodeBlockItem struct {
	execution.Block
	Lang            string   `json:"lang,omitempty"`
	AnnotationBlock []string `json:"annotation_block,omitempty"`
}

// TOCItem is a table-of-contents item
type TOCItem struct {
	Anchor string `json:"anchor"`
	Header string `json:"header"`
}

// Block is a rendered block that can be displayed by the frontend
type Block struct {
	TableOfContents []TOCItem       `json:"toc,omitempty"`
	Description     string          `json:"description,omitempty"`
	CodeBlock       []CodeBlockItem `json:"code_block,omitempty"`
}

// Link is link-related information
type Link struct {
	Raw string
	Sym pythonresource.Symbol
}

// Rendered is the top-level rendered contents
type Rendered struct {
	Title   string  `json:"title"`
	Content []Block `json:"content"`
	Links   []Link  `json:"-"`
}

// Render renders the provided markdown as a sequence of RenderedBlocks
func Render(ctx kitectx.Context, sandbox execution.Manager, rm pythonresource.Manager, raw Raw) (Rendered, error) {
	// reserve space for headline & TOC
	out := Rendered{
		Title:   raw.title,
		Content: []Block{{}, {}},
	}

	var tocItems []TOCItem
	var allErrs errors.Errors

	src := raw.after
	for len(src) > 0 {
		var before, info, code []byte
		before, info, code, src = splitOnCodeBlock(src)

		var newItems []TOCItem
		before, newItems = anchorHeaders(before, raw.Headers)
		before, out.Links = link(out.Links, rm, before)
		tocItems = append(tocItems, newItems...)

		if len(before) > 0 {
			out.Content = append(out.Content, Block{Description: string(before)})
		}

		if len(code) == 0 {
			continue
		}

		// code
		lang := string(info)
		var blocks []execution.Block
		var errs errors.Errors
		switch lang {
		case "python":
			spec, code, err := extractExecution(code, raw)
			errs = errors.Append(errs, err)
			blocks, err = sandbox.Run(ctx, spec, code)
			errs = errors.Append(errs, err)
		default:
			for _, line := range strings.Split(strings.TrimRight(string(code), "\n"), "\n") {
				refline := strings.TrimRightFunc(line, unicode.IsSpace)
				blocks = append(blocks, execution.Block{
					CodeLine: &refline,
				})
			}
		}

		var items []CodeBlockItem
		if errs != nil {
			var i CodeBlockItem
			i.Output = &execution.Output{
				Type:  "text",
				Title: "!! execution errors",
				Data:  errs.Error(),
			}
			items = append(items, i)
		}

		items = append(items, annotateBlocks(blocks, lang)...)
		out.Content = append(out.Content, Block{CodeBlock: items})
		allErrs = errors.Append(allErrs, errs)
	}

	if raw.title != "" {
		out.Content[0] = Block{Description: "# " + raw.title}
	}
	if len(tocItems) > 1 {
		out.Content[1] = Block{TableOfContents: tocItems}
	}

	return out, allErrs
}
