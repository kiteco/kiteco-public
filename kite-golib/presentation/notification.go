package presentation

// Notification ...
type Notification struct {
	Title   string   `json:"title"`
	Body    string   `json:"body"`
	Buttons []Button `json:"buttons"`
}

// Button should be created with a link only if the
// Action is Open. Due to lack of sum types in Go 1,
// espressing this in code seems to obfuscate more than help.
type Button struct {
	Text   string  `json:"text"`
	Action Action  `json:"action"`
	Link   *string `json:"link,omitempty"`
}

// Action ...
type Action string

// Action enum
const (
	Open    Action = "open"
	Dismiss        = "dismiss"
)
