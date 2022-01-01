package githubdata

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/require"
)

func Test_ExtractedSamples_Simple(t *testing.T) {
	// No change, no sites
	src := "a\nb\nc\nd"
	dst := "a\nb\nc\nd"
	sites := extractSites(t, src, dst)
	require.Empty(t, sites)

	src = "a\nb\nc\nd"
	dst = "a\nBd\nc\nd"
	sites = extractSites(t, src, dst)
	assertValidSites(t, src, dst, sites)

	src = "a\nc\nd"
	dst = "a\nb\nc\nd"
	sites = extractSites(t, src, dst)
	assertValidSites(t, src, dst, sites)

	src = "a\nb\nc\nd"
	dst = "a\nb\nd"
	sites = extractSites(t, src, dst)

	// Insertion at the end
	src = "a\nb\nc\nd"
	dst = "a\nb\nc\nd\ne"
	sites = extractSites(t, src, dst)
	assertValidSites(t, src, dst, sites)

	// Only insertions
	src = ""
	dst = "a\nb\nc\nd"
	sites = extractSites(t, src, dst)
	assertValidSites(t, src, dst, sites)
}

func Test_ExtractSamples_Detailed(t *testing.T) {
	src := "a\nb\nc\nd"
	dst := "a\nBd\nc\nDb"
	sites := extractSites(t, src, dst)
	assertValidSites(t, src, dst, sites)

	require.Len(t, sites, 2)
	require.Equal(t, "Bd\n", sites[0].DstWindow)
	require.Equal(t, "b\n", sites[0].SrcWindow)
	require.Equal(t, "Db", sites[1].DstWindow)
	require.Equal(t, "d", sites[1].SrcWindow)
}

func assertValidSites(t *testing.T, src, dst string, sites []PredictionSite) {
	require.NotEmpty(t, sites)
	for _, site := range sites {
		srcBuffer := &bytes.Buffer{}
		srcBuffer.WriteString(site.SrcContextBefore)
		srcBuffer.WriteString(site.SrcWindow)
		srcBuffer.WriteString(site.SrcContextAfter)
		require.Equal(t, src, srcBuffer.String())

		dstBuffer := &bytes.Buffer{}
		dstBuffer.WriteString(site.DstContextBefore)
		dstBuffer.WriteString(site.DstWindow)
		dstBuffer.WriteString(site.DstContextAfter)
		require.Equal(t, dst, dstBuffer.String())
	}
}

func extractSites(t *testing.T, src, dst string) []PredictionSite {
	opts := Options{}
	extractor, err := NewExtractor(opts)
	require.NoError(t, err)
	diffs, err := computeDiffs(src, dst)
	require.NoError(t, err)
	return extractor.extractSamplesFromDiff(makePullRequest(), "test.py", diffs)
}

func makePullRequest() *github.PullRequest {
	number := 42
	closedAt := time.Now()
	return &github.PullRequest{
		Number:   &number,
		ClosedAt: &closedAt,
	}
}
