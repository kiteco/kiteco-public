package pythonlocal

import "github.com/kiteco/kiteco/kite-go/response"

// DocumentationResponse converts a pythonlocal.Documentation to a response.Documentation
func DocumentationResponse(doc *Documentation) *response.PythonDocumentation {
	structuredDoc := &response.PythonStructuredDoc{
		Ident:       doc.Identifier,
		Description: doc.HTML,
	}
	return &response.PythonDocumentation{
		// Get full name
		ID:            doc.CanonicalName,
		Name:          doc.Identifier,
		Type:          response.PythonDocumentationType,
		LocalCode:     true,
		Description:   doc.Description,
		StructuredDoc: structuredDoc,
	}
}

// DefinitionPathForFile converts our internal representation
// of a path into a native path on the user's filesystem.
func DefinitionPathForFile(file string) (string, error) {
	return fromUnix(file)
}
