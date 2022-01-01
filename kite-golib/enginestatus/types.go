package enginestatus

import "github.com/kiteco/kiteco/kite-golib/presentation"

// Response encapsulates a Kite Status response.
type Response struct {
	Status string               `json:"status"`
	Short  string               `json:"short"`
	Long   string               `json:"long"`
	Button *presentation.Button `json:"button,omitempty"`
}
