package main

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("user-mux")

	badGatewayCounter  = section.Counter("Bad Gateway")
	badGatewayDuration = section.SampleDuration("Bad Gateway")
	handlerDuration    = section.SampleDuration("All requests")

	responseCodes      = section.Breakdown("Response codes")
	eventResponseCodes = section.Breakdown("Event response codes")
)

func init() {
	responseCodes.AddCategories("1xx", "2xx", "3xx", "4xx", "5xx", "other")
	eventResponseCodes.AddCategories("1xx", "2xx", "3xx", "4xx", "5xx", "other")

	badGatewayCounter.Headline = true
	handlerDuration.Headline = true
}
