package pythondocs

import (
	"strings"

	"github.com/kiteco/kiteco/kite-go/response"
)

const (
	builtin = "builtins."
)

// DocumentationResponse takes a language entity and builds a PythonDocumentation response.
func DocumentationResponse(r *Result) *response.PythonDocumentation {
	// If entity is not found then we still want to show children/ancestors, so build an empty entity
	entity := r.Entity
	if entity == nil {
		entity = MakeEmptyEntity(r.Ident, r.Kind)
	}

	return &response.PythonDocumentation{
		ID:            entity.FullIdent(),
		Name:          entity.Name(),
		Type:          response.PythonDocumentationType,
		Kind:          entity.Kind.String(),
		Description:   entity.Doc,
		Signature:     strings.TrimPrefix(entity.Signature, builtin),
		StructuredDoc: structuredDocResponse(entity.StructuredDoc),
		Ancestors:     identifierResponses(r.Ancestors),
		Children:      identifierResponses(r.Children),
		References:    identifierResponses(r.References),
	}
}

// identifierResponses
func identifierResponses(idents []Identifier) []response.PythonIdentifier {
	// use empty slice literal not nil slice to avoid "null" in json
	out := []response.PythonIdentifier{}
	for _, in := range idents {
		out = append(out, response.PythonIdentifier{
			ID:         in.Ident,
			RelativeID: in.Rel,
			Name:       in.Name,
			Kind:       in.Kind.String(),
		})
	}
	return out
}

// structuredDocResponse takes a StructuredDoc and builds a PythonStructuredDoc response.
func structuredDocResponse(structuredDoc *StructuredDoc) *response.PythonStructuredDoc {
	if structuredDoc == nil {
		return nil
	}
	resp := &response.PythonStructuredDoc{
		Ident:       strings.TrimPrefix(structuredDoc.Ident, builtin),
		Description: structuredDoc.DescriptionHTML,
		ReturnType:  structuredDoc.ReturnType,
	}
	for _, param := range structuredDoc.Parameters {
		name := param.Name
		switch param.Type {
		case VarParamType:
			name = "*" + name
		case VarKwParamType:
			name = "**" + name
		}
		resp.Parameters = append(resp.Parameters, &response.PythonParameter{
			Type:        param.Type.String(),
			Name:        name,
			Default:     param.Default,
			Description: param.DescriptionHTML,
		})
	}
	return resp
}

// MakeEmptyEntity creates an empty LangEntity for the given name and kind.
func MakeEmptyEntity(ident string, kind LangEntityKind) *LangEntity {
	const noDocAvailable = "No documentation available"
	var sel string
	parts := strings.Split(ident, ".")
	if len(parts) > 1 {
		ident = strings.Join(parts[:len(parts)-1], ".")
		sel = parts[len(parts)-1]
	}
	return &LangEntity{
		Module: parts[0],
		Ident:  ident,
		Sel:    sel,
		Kind:   kind,
		Doc:    noDocAvailable,
		StructuredDoc: &StructuredDoc{
			Ident:           ident,
			DescriptionHTML: "<body><p>" + noDocAvailable + "</p></body>",
		},
	}
}
