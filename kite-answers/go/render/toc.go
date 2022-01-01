package render

import (
	"bytes"
	"fmt"
	"regexp"
)

/*
  Headers to be considered must be of the form
  ## abc {#xyz}
*/
var h2RE = regexp.MustCompile(`(?m)^ {0,3}## +([^\r\n]*) +{#(.+)}(?: #+)? *\r?$`)

/*
  Adds anchors to any 'h2's in the input.
  Also updates the TOC map with the header name
  if no shortened name is provided by the author.
*/
func anchorHeaders(input []byte, headerMap map[string]string) ([]byte, []TOCItem) {
	var items []TOCItem
	var newBuf bytes.Buffer

	curIdx := 0
	for _, indices := range h2RE.FindAllSubmatchIndex(input, -1) {
		if indices == nil {
			continue
		}

		headerName := string(input[indices[2]:indices[3]])
		anchorName := string(input[indices[4]:indices[5]])

		// Linkify header line and update header map
		item := TOCItem{
			Anchor: anchorName,
			Header: headerName,
		}
		if _, ok := headerMap[anchorName]; ok {
			item.Header = headerMap[anchorName]
		}
		items = append(items, item)

		newBuf.Write(input[curIdx:indices[0]])
		newBuf.Write(transformHeader(anchorName, headerName))
		curIdx = indices[1]
	}
	newBuf.Write(input[curIdx:len(input)])

	return newBuf.Bytes(), items
}

func transformHeader(anchorName string, headerName string) []byte {
	return []byte(fmt.Sprintf("## %s <a name=\"%s\"></a>", headerName, anchorName))
}
