package tracks

var (
	// Bucket is the S3 bucket containing segment events
	Bucket = "kite-segment-backend-http-requests"
	// MetricsBucket is the S3 bucket containing kite_status segment events
	MetricsBucket = "kite-metrics"
)

var (
	// ClientEventSource contains all the client HTTP requests + key events
	ClientEventSource = "gw5gdCNLOO"
	// OldClientEventSource contains all the client HTTP requests + key events
	OldClientEventSource = "64QM51qPD1"
	// StatusEventSource contains all kite_status events
	StatusEventSource = "Rgn399rf0J"
)
