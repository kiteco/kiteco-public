package desktoplogin

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section                 = status.NewSection("client/internal/desktoplogin")
	usedCounterDistribution = section.CounterDistribution("Desktop login used")
)
