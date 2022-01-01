package segmentsrc

// Source encapsulates a Segment "source" along with the canonical S3 destination for that source
type Source struct {
	Bucket   string
	Region   string
	SourceID string
	WriteKey string
}

var (
	// Production is a Source
	Production = Source{
		Bucket:   "kite-metrics",
		Region:   "us-east-1",
		SourceID: "XXXXXXX",
		WriteKey: "XXXXXXX",
	}
	// CalleeTracking is a Source
	CalleeTracking = Source{
		Bucket:   "kite-segment-callee-tracking",
		Region:   "us-west-1",
		SourceID: "XXXXXXX",
		WriteKey: "XXXXXXX",
	}
	// CompletionsTracking is a Source
	CompletionsTracking = Source{
		Bucket:   "kite-segment-completions-tracking",
		Region:   "us-west-1",
		SourceID: "XXXXXXX",
		WriteKey: "XXXXXXX",
	}
	// KiteService is a Source
	KiteService = Source{
		Bucket:   "kite-segment-kite-service",
		Region:   "us-west-1",
		SourceID: "IavDfjeu6y",
		WriteKey: "XXXXXXX",
	}
	// ClientEventsTrimmed is a Source
	ClientEventsTrimmed = Source{
		Bucket:   "kite-segment-backend-http-requests",
		Region:   "us-west-1",
		SourceID: "XXXXXXX",
		WriteKey: "XXXXXXX",
	}
)
