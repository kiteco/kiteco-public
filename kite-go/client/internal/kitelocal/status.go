package kitelocal

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section   = status.NewSection("client/internal/kitelocal")
	userCount = section.Counter("Users")
)

func init() {
	userCount.Set(1)
}
