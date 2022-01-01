package pythondocs

import (
	"fmt"
	"html"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

const (
	// DefaultSchemaPath is the default path to the .xsd file
	DefaultSchemaPath = "s3://kite-emr/datasets/documentation/html-format/2016-01-14_13-55-00-PM/dochtml.xsd"
	rootDepth         = 0
)

var (
	ignoreRoles = []string{"copyright", "license"}
)

// HTMLNormalizer contains logic to convert HTML into our custom HTML schema.
type HTMLNormalizer struct {
	Graph   *pythonimports.Graph
	Parents ParentMap
}

// NewHTMLNormalizer constructs a new HTMLNormalizer object.
func NewHTMLNormalizer(graph *pythonimports.Graph) *HTMLNormalizer {
	return &HTMLNormalizer{
		Graph:   graph,
		Parents: BuildParentMap(graph.Nodes),
	}
}

func (n *HTMLNormalizer) isTextNode(f *goquery.Selection) bool {
	copy := f.Clone()
	original := copy.Text()
	copy.WrapHtml("<div></div>")
	wrapped, err := copy.Parent().Html()
	if err != nil {
		log.Fatalf("Error while getting html")
	}
	return original == html.UnescapeString(wrapped)
}

func (n *HTMLNormalizer) parseParams(s *goquery.Selection, params map[string]string) {
	s.Children().Each(func(j int, f *goquery.Selection) {
		if f.Is("table") {
			var currParam string
			f.Find("tr").Each(func(i int, g *goquery.Selection) {
				firstCol := g.Find("td").First()
				secondCol := firstCol.Next()

				if text := firstCol.Text(); text != "" && strings.HasPrefix(text, "param ") && strings.HasSuffix(text, ":") {
					split := strings.Split(text, " ")
					currParam = strings.TrimRight(split[1], ":")
				}
				if currParam != "" && secondCol.Is("td") && secondCol.Text() != "" {
					desc, _ := secondCol.Html()
					params[currParam] = desc
					currParam = ""
				}
			})
		}
	})
}

func (n *HTMLNormalizer) parseNormalizedDescription(s *goquery.Selection, ident string, depth int) string {
	var description string
	s.Children().Each(func(j int, f *goquery.Selection) {
		if segment := n.parseNormalizedSegment(f, ident, depth); segment != "" {
			description += segment
			return
		}
		if f.HasClass("section") {
			description += n.parseNormalizedDescription(f, ident, depth+1)
		}
	})
	if depth == rootDepth && len(description) > 0 {
		description = fmt.Sprintf("<body>%s</body>", description)
	}
	return description
}

func (n *HTMLNormalizer) parseNormalizedSegment(f *goquery.Selection, ident string, depth int) string {
	if f.Is("p") || f.Is("dd") || f.Is("strong") || f.Is("em") || n.isHeader(f) {
		tag := f.Get(0).Data
		if tag == "h7" || tag == "h8" {
			tag = "h6"
		}
		// capitalize first letter of contents in a child <p>...</p>
		if f.Is("p") && depth == rootDepth {
			first := f.Contents().First()
			if n.isTextNode(first) {
				first.ReplaceWithHtml(capitalizeFirstLetter(first.Text()))
			}
		}

		html := fmt.Sprintf("<%s>", tag)
		f.Contents().Each(func(j int, g *goquery.Selection) {
			html += n.parseNormalizedSegment(g, ident, depth+1)
		})
		html += fmt.Sprintf("</%s>", tag)
		return html
	} else if f.Is("dl") {
		var html string
		if cls, ok := f.Attr("class"); ok {
			if strings.Contains(cls, "docutils") {
				cls = "plainlist"
			} else if strings.Contains(cls, "method") {
				cls = "method"
			}
			html += `<dl class="` + cls + `">`
		} else {
			html += "<dl>"
		}
		f.Contents().Each(func(j int, g *goquery.Selection) {
			if g.Is("dt") || g.Is("dd") {
				html += n.parseNormalizedSegment(g, ident, depth+1)
			}
		})
		html += "</dl>"
		return html
	} else if f.Is("dt") {
		var html string
		if id, ok := f.Attr("id"); ok {
			html += `<dt id="` + id + `">`
		} else {
			html += "<dt>"
		}
		f.Contents().Each(func(j int, g *goquery.Selection) {
			html += n.parseNormalizedSegment(g, ident, depth+1)
		})
		html += "</dt>"
		return html
	} else if f.Is("table") {
		if f.HasClass("docutils") {
			var ignore bool
			f.Find("tr").Each(func(i int, g *goquery.Selection) {
				if th := g.Find("th").First(); th.HasClass("field-name") {
					for _, ignored := range ignoreRoles {
						if strings.Contains(th.Text(), ignored) {
							ignore = true
						}
					}
				}
			})
			if ignore {
				return ""
			}
		}
		html := "<table>"
		f.Contents().Each(func(j int, g *goquery.Selection) {
			if g.Is("thead") || g.Is("tbody") {
				html += n.parseNormalizedSegment(g, ident, depth+1)
			} else if g.Is("tr") {
				html += "<tr>"
				g.Contents().Each(func(k int, h *goquery.Selection) {
					if h.Is("td") {
						html += "<td>"
						html += n.parseNormalizedSegment(h, ident, depth+1)
						html += "</td>"
					} else if h.Is("th") {
						html += "<th>"
						html += n.parseNormalizedSegment(h, ident, depth+1)
						html += "</th>"
					}
				})
				html += "</tr>"
			}
		})
		html += "</table>"
		return html
	} else if f.Is("thead") || f.Is("tbody") {
		var html string
		f.Find("tr").Each(func(j int, g *goquery.Selection) {
			if valign, ok := f.Attr("valign"); ok {
				html += `<tr valign="` + valign + `">`
			} else {
				html += "<tr>"
			}
			g.Contents().Each(func(k int, h *goquery.Selection) {
				if h.Is("th") || h.Is("td") {
					if f.Is("thead") {
						html += "<th>"
					} else if f.Is("tbody") {
						html += "<td>"
					}
					html += n.parseNormalizedSegment(h, ident, depth+1)
					if f.Is("thead") {
						html += "</th>"
					} else if f.Is("tbody") {
						html += "</td>"
					}
				}
			})
			html += "</tr>"
		})
		return html
	} else if f.Is("th") || f.Is("td") {
		var html string
		f.Contents().Each(func(j int, g *goquery.Selection) {
			html += n.parseNormalizedSegment(g, ident, depth+1)
		})
		return html
	} else if f.Is("cite") {
		return "<i>" + f.Text() + "</i>"
	} else if f.Is("span") && n.isRole(f) {
		title, target := n.resolveRole(f.Text(), ident)
		if target == "" {
			return fmt.Sprintf("<a>%s</a>", title)
		}
		return fmt.Sprintf("<a href=\"#%s\" class=\"internal_link\">%s</a>", target, title)
	} else if f.Is("span") {
		html := "<span"
		if f.HasClass("versionmodified") && f.Parent().Is("p") {
			if class, ok := f.Parent().Attr("class"); ok && (class == "versionadded" || class == "versionchanged" || class == "deprecated") {
				html += ` class="` + class + `">`
			} else {
				html += ">"
			}
		} else {
			html += ">"
		}
		f.Contents().Each(func(j int, g *goquery.Selection) {
			html += n.parseNormalizedSegment(g, ident, depth+1)
		})
		html += "</span>"
		return html
	} else if f.Is("a") {
		if f.Text() == `¶` {
			return ""
		}
		html := "<a"
		if link, ok := f.Attr("href"); ok {
			if f.HasClass("internal") {
				if idx := strings.Index(link, "#"); idx != -1 {
					link = link[idx+1:]
					if strings.Contains(link, "-") {
						link = ""
					} else if _, ok := n.resolveName(link, ident); !ok {
						link = ""
					} else {
						html += ` class="internal_link"`
					}
				}
			} else {
				html += ` class="external_link"`
			}
			if link != "" {
				html += fmt.Sprintf(" href=\"#%s\"", link)
			}
		}
		html += ">"
		f.Contents().Each(func(j int, g *goquery.Selection) {
			if g.Is("span") {
				g.Contents().Each(func(k int, h *goquery.Selection) {
					html += n.parseNormalizedSegment(h, ident, depth+1)
				})
			} else {
				html += n.parseNormalizedSegment(g, ident, depth+1)
			}
		})
		html += "</a>"
		return html
	} else if f.Is("tt") || f.Is("code") {
		var str string
		f.Contents().Each(func(j int, g *goquery.Selection) {
			if (g.Is("span") && g.HasClass("pre")) || g.Is("em") || n.isTextNode(g) {
				str += "<code>" + html.EscapeString(removeInvalidUnicode(g.Text())) + "</code>"
			}
		})
		return str
	} else if f.Is("div") || f.Is("blockquote") {
		var html string
		var removeNext bool
		f.Contents().Each(func(j int, g *goquery.Selection) {
			if removeNext {
				removeNext = false
				return
			} else if n.isMainDiv(f) {
				// remove header that repeats the identifier name eg. flask
				if j == 1 && n.isHeader(g) && strings.Contains(strings.ToLower(strings.TrimSpace(g.Text())), strings.ToLower(ident)) {
					// remove subsequent spaces
					if next := f.Contents().Eq(j + 1); next.Length() > 0 && n.isTextNode(next) && len(strings.TrimSpace(next.Text())) == 0 {
						removeNext = true
					}
					return
				}
			}
			html += n.parseNormalizedSegment(g, ident, depth+1)
		})
		return html
	} else if f.Is("pre") {
		var contents string
		f.Contents().Each(func(j int, g *goquery.Selection) {
			if g.Is("span") || n.isTextNode(g) || g.Is("code") {
				contents += g.Text()
			}
		})
		return "<pre class=\"lang-python\"><code>" + html.EscapeString(removeInvalidUnicode(contents)) + "</code></pre>"
	} else if f.Is("ul") || f.Is("ol") {
		return n.parseNormalizedList(f, ident, depth)
	} else if f.Is("li") {
		return n.parseNormalizedUnorderedListItem(f, ident, depth)
	} else if f.Is("big") {
		return f.Text()
	} else if n.isTextNode(f) {
		return html.EscapeString(removeInvalidUnicode(f.Text()))
	}
	return ""
}

func (n *HTMLNormalizer) isRole(f *goquery.Selection) bool {
	return f.HasClass("func") || f.HasClass("class") || f.HasClass("meth") || f.HasClass("mod") || f.HasClass("attr") || f.HasClass("data") || f.HasClass("const") || f.HasClass("exc") || f.HasClass("obj") || f.HasClass("ref")
}

func (n *HTMLNormalizer) isHeader(f *goquery.Selection) bool {
	return f.Is("h1") || f.Is("h2") || f.Is("h3") || f.Is("h4") || f.Is("h5") || f.Is("h6") || f.Is("h7") || f.Is("h8")
}

func (n *HTMLNormalizer) isMainDiv(f *goquery.Selection) bool {
	return f.Is("div") && f.HasClass("document")
}

func (n *HTMLNormalizer) parseNormalizedList(s *goquery.Selection, ident string, depth int) string {
	var list string
	if s.Is("ul") {
		list += "<ul>"
	} else if s.Is("ol") {
		list += "<ol>"
	}
	s.Children().Each(func(j int, f *goquery.Selection) {
		if !f.Is("li") {
			log.Println("Found non-li in ul/ol")
			return
		}
		list += n.parseNormalizedSegment(f, ident, depth+1)
	})
	if s.Is("ul") {
		list += "</ul>"
	} else if s.Is("ol") {
		list += "</ol>"
	}
	return list
}

func (n *HTMLNormalizer) parseNormalizedUnorderedListItem(s *goquery.Selection, ident string, depth int) string {
	item := "<li>"
	s.Contents().Each(func(j int, g *goquery.Selection) {
		item += n.parseNormalizedSegment(g, ident, depth+1)
	})
	item += "</li>"
	return item
}

// Resolve roles that appear in docstrings by using the import graph.
// Adapted from the `find_obj` method in the Sphinx source:
// https://github.com/sphinx-doc/sphinx/blob/6b7b51a55aa0dc419d9fd8dae17bbec197bd2724/sphinx/domains/python.py
//
// Also refer to description of how Sphinx resolves roles here:
// http://www.sphinx-doc.org/en/stable/domains.html#python-roles
func (n *HTMLNormalizer) resolveRole(fn, ident string) (title, target string) {
	if strings.HasSuffix(fn, "()") {
		fn = fn[:len(fn)-2]
	}
	if strings.HasPrefix(fn, "~.") {
		resolved, ok := n.resolveRelative(fn[2:], ident)
		if !ok {
			resolved, ok = n.resolveSuffix(fn[2:], ident)
		}
		if ok {
			target = resolved
			title = n.selector(resolved)
		}
	} else if strings.HasPrefix(fn, "~") {
		target = fn[1:]
		title = n.selector(fn[1:])
	} else if strings.HasPrefix(fn, ".") {
		resolved, ok := n.resolveRelative(fn[1:], ident)
		if !ok {
			resolved, ok = n.resolveSuffix(fn[1:], ident)
		}
		if ok {
			target = resolved
			title = resolved
		}
	} else {
		resolved, ok := n.resolveName(fn, ident)
		if !ok {
			resolved, ok = n.resolveSuffix(fn, ident)
		}
		if ok {
			target = resolved
			title = n.selector(resolved)
		}
	}
	title = strings.TrimLeft(title, "_")
	target = strings.TrimLeft(target, "_")
	if title == "" && target == "" {
		log.Println("could not resolve", fn, "in", ident)
		title = strings.TrimLeft(fn, "~")
		title = strings.TrimLeft(title, ".")
		target = ""
	}
	return
}

func (n *HTMLNormalizer) resolveName(name, ident string) (string, bool) {
	if node, err := n.Graph.Find(name); err == nil { // first exact match
		return n.nodeName(node, ""), true
	}
	split := strings.Split(ident, ".")
	for i := 1; i < len(split); i++ { // most general to most specific
		newName := fmt.Sprintf("%s.%s", strings.Join(split[:i], "."), name)
		if node, err := n.Graph.Find(newName); err == nil {
			return n.nodeName(node, ""), true
		}
	}
	return "", false
}

func (n *HTMLNormalizer) resolveRelative(name, ident string) (string, bool) {
	split := strings.Split(ident, ".")
	for i := len(split); i > 0; i-- { // most specific to most general
		newName := fmt.Sprintf("%s.%s", strings.Join(split[:i], "."), name)
		if node, err := n.Graph.Find(newName); err == nil {
			return n.nodeName(node, ""), true
		}
	}
	if node, err := n.Graph.Find(name); err == nil { // finally exact match
		return n.nodeName(node, ""), true
	}
	return "", false
}

func (n *HTMLNormalizer) resolveSuffix(suffix, ident string) (string, bool) {
	module := strings.Split(ident, ".")[0]
	suffix = "." + suffix
	var resolved string
	var found, ambiguous bool
	walker := func(name string, node *pythonimports.Node) bool {
		if ambiguous {
			return false
		}
		if canonical := n.nodeName(node, suffix); canonical != "" {
			if !found {
				resolved = canonical
				found = true
			} else {
				ambiguous = true
			}
		}
		return !ambiguous
	}
	err := n.Graph.Walk(module, walker)
	if err != nil {
		log.Println("error walking graph:", err)
		return "", false
	}
	if ambiguous {
		return "", false
	}
	return resolved, found
}

func (n *HTMLNormalizer) selector(name string) string {
	if len(name) > 0 {
		split := strings.Split(name, ".")
		return split[len(split)-1]
	}
	return ""
}

func (n *HTMLNormalizer) nodeName(node *pythonimports.Node, suffix string) string {
	var names []string
	if !node.CanonicalName.Empty() {
		names = append(names, node.CanonicalName.String())
	}
	names = append(names, n.namesFromParents(node)...)

	for _, name := range names {
		if strings.HasSuffix(name, suffix) {
			return name
		}
	}
	return ""
}

func (n *HTMLNormalizer) namesFromParents(node *pythonimports.Node) []string {
	var names []string
	for _, parent := range n.Parents[node] {
		if !parent.CanonicalName.Empty() {
			for member, child := range parent.Members {
				if child != nil && child.ID == node.ID {
					full := parent.CanonicalName.String() + "." + member
					names = append(names, full)
				}
			}
		}
	}
	return names
}

func capitalizeFirstLetter(s string) string {
	if s == "" {
		return s
	}
	r, sz := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	if unicode.IsLetter(r) {
		r = unicode.ToUpper(r)
	}
	return string(r) + s[sz:]
}

func isLetter(r int) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// --

func removeInvalidUnicode(htmlStr string) string {
	var ret string
	for _, c := range htmlStr {
		switch c {
		case rune('\u00B6'), rune('\u0007'), rune('\u0008'):
		default:
			ret += string(c)
		}
	}
	return ret
}
