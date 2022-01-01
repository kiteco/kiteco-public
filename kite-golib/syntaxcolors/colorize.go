//go:generate go-bindata -o bindata.go -pkg syntaxcolors syntaxcolors.css

package syntaxcolors

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"

	"github.com/sourcegraph/syntaxhighlight"
)

// CursorPrinter implements syntaxhighlight.Printer to provide cursor insertion
type cursorPrinter struct {
	cursor  int
	count   int
	printer syntaxhighlight.HTMLPrinter
}

// Print implements syntaxhighlight.Printer.Print in order to insert a span at the cursor position.
func (p *cursorPrinter) Print(w io.Writer, kind syntaxhighlight.Kind, text string) error {
	i := p.cursor - p.count
	if i >= 0 && i <= len(text) {
		// Insert a marker
		text = text[:i] + "$" + text[i:]
		var buf bytes.Buffer

		// Apply syntax highlighting
		err := p.printer.Print(&buf, kind, text)
		if err != nil {
			return err
		}

		// Replace marker with span (which would have been escaped if we had put it in directly)
		b := bytes.Replace(buf.Bytes(), []byte("$"), []byte(`<span class="cursor"></span>`), -1)
		_, err = w.Write(b)
		if err != nil {
			return err
		}
	} else {
		p.printer.Print(w, kind, text)
	}

	p.count += len(text)
	return nil
}

// Colorize applies syntax highlighting to the given code chunk, and if cursor is not -1 then insert
// a cursor at that position.
func Colorize(code []byte, cursor int) template.HTML {
	printer := &cursorPrinter{
		cursor:  cursor,
		printer: syntaxhighlight.HTMLPrinter(syntaxhighlight.DefaultHTMLConfig),
	}

	var buf bytes.Buffer
	err := syntaxhighlight.Print(syntaxhighlight.NewScanner(code), &buf, printer)
	if err != nil {
		log.Println(err)
		// Just return the original code un-highlighted
		return template.HTML(code)
	}

	return template.HTML(buf.Bytes())
}

// DefaultStylesheet returns a buffer containing the default syntax coloring stylehseet
func DefaultStylesheet() template.CSS {
	return template.CSS(MustAsset("syntaxcolors.css"))
}

// HandleDefaultStylesheet is a function
func HandleDefaultStylesheet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", mime.TypeByExtension(".css"))
	w.Write([]byte(DefaultStylesheet()))
}
