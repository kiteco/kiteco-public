package pythondocs

import (
	"fmt"
	"os"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

const (
	coverageStyle = `
<style>
.kite_highlight {
    background-color:#CCFFCC;
}
.kite_header {
    color: white;
    font-family: Helvetica, Arial, Sans-Serif;
    background-color: #006600;
}
</script>`
)

func renderCoverage(doc *goquery.Document, module string) error {
	doc.Find("head").AppendHtml(coverageStyle)

	f, err := os.Create(fmt.Sprintf("%s-coverage.html", module))
	if err != nil {
		return err
	}
	defer f.Close()
	if err := html.Render(f, doc.Selection.Nodes[0]); err != nil {
		return fmt.Errorf("error rendering template: %v", err)
	}
	return nil
}
