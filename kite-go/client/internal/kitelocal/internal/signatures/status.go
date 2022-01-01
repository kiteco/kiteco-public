package signatures

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("client/internal/kitelocal/internal/signatures")

	signaturesCount     = section.Counter("Signatures endpoint hit")
	signaturesCountDist = section.CounterDistribution("Signatures endpoint hit")

	signaturesDuration = section.SampleDuration("signatures.handleSignatures")

	signaturesReturnedCount     = section.Counter("Signatures returned")
	signaturesReturnedCountDist = section.CounterDistribution("Signatures returned")

	aggregateHitRate     = section.Ratio("Signatures aggregate hit rate")
	aggregateHitRateDist = section.RatioDistribution("Signatures aggregate hit rate")
)

func init() {
	signaturesDuration.SetSampleRate(1.0)
}
