package pythondocs

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// AddNoFollow adds `nofollow` to the <a class="internal_link" href="#{id}"> tags if the ID matches the provided predicate.
// This is unused, and can be removed whenever someone next sees this message.
// Leaving it in for now (8/6/2019), in case we want to use this, since it's not trivial to reproduce.
func AddNoFollow(htext string, pred func(id string) bool) string {
	z := html.NewTokenizer(strings.NewReader(htext))
	var out bytes.Buffer
	for {
		ty := z.Next()
		if ty == html.ErrorToken {
			if z.Err() == io.EOF {
				return out.String()
			}
			// TODO(naman) log? rollbar?
			return htext
		}

		raw := z.Raw()
		if ty == html.StartTagToken {
			name, hasAttr := z.TagName()
			if strings.ToLower(string(name)) == "a" && hasAttr {
				var checkNoFollow bool
				var href string
				for {
					k, v, more := z.TagAttr()
					switch strings.ToLower(string(k)) {
					case "class":
						if strings.ToLower(string(v)) == "internal_link" {
							checkNoFollow = true
						}
					case "href":
						href = string(v)
					default:
						// ignore; TODO(naman) log? rollbar?
					}
					if !more {
						break
					}
				}

				if checkNoFollow {
					if pred(strings.TrimPrefix(href, "#")) {
						fmt.Fprintf(&out, "<a class=\"internal_link\" href=\"%s\" rel=\"nofollow\">", href)
						continue
					}
				}
			}
		}
		out.Write(raw)
	}
}
