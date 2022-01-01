package permissions

import "github.com/kiteco/kiteco/kite-go/client/component"

var supportMap = map[string]component.SupportStatus{
	".c": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".cc": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".cpp": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".cs": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".css": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".go": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".h": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".hpp": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".html": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".java": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".js": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".jsx": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".kt": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".less": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".m": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".php": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".py": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
		HoverSupported:       true,
		SignaturesSupported:  true,
	},
	".pyw": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
		HoverSupported:       true,
		SignaturesSupported:  true,
	},
	".rb": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".scala": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".sh": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".ts": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".tsx": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
	".vue": component.SupportStatus{
		EditEventSupported:   true,
		CompletionsSupported: true,
	},
}
