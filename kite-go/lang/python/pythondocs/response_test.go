package pythondocs

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/response"
)

var (
	resp  *response.PythonDocumentation
	resps []*response.PythonDocumentation
)

func BenchmarkDocumentationResponse(b *testing.B) {
	graph := pythonimports.MockGraph("suds.xsd.sxbase.PartElement.content")
	corpus := MockDocumentation(graph, "suds.xsd.sxbase.PartElement.content")
	entity, found := corpus.FindIdent("suds.xsd.sxbase.PartElement.content")
	if !found {
		b.Error("benchmark entity not found")
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DocumentationResponse(entity)
	}
}
