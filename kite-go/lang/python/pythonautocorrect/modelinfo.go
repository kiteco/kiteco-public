package pythonautocorrect

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
)

// ModelInfo for the correcter.
func (c *Correcter) ModelInfo(v uint64) (editorapi.AutocorrectModelInfoResponse, error) {
	if version != v {
		return editorapi.AutocorrectModelInfoResponse{}, fmt.Errorf("unsupported version %d", v)
	}

	example := editorapi.AutocorrectExample{
		Synopsis: "Kite will fix missing colons.",
		Old: []editorapi.AutocorrectExampleLine{
			editorapi.AutocorrectExampleLine{
				Text: "def foo()",
			},
		},
		New: []editorapi.AutocorrectExampleLine{
			editorapi.AutocorrectExampleLine{
				Text: "def foo():",
				Emphasis: []editorapi.AutocorrectLineEmphasis{
					editorapi.AutocorrectLineEmphasis{
						StartBytes: 9,
						StartRunes: 9,
						EndBytes:   10,
						EndRunes:   10,
					},
				},
			},
		},
	}

	return editorapi.AutocorrectModelInfoResponse{
		DateShipped: 1257894000,
		Examples: []editorapi.AutocorrectExample{
			example,
		},
	}, nil
}
