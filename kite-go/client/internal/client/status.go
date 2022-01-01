package client

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section       = status.NewSection("client/internal/kite")
	userCount     = section.Counter("Users")
	skipBreakdown = section.Breakdown("Skip event reasons")
)

func init() {
	userCount.Set(1)
	skipBreakdown.AddCategories("unsaved", "unsupported file", "not whitelisted", "file too large", "editor skip")
}
