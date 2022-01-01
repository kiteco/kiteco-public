package event

import (
	"time"

	"github.com/kiteco/kiteco/kite-go/response"
)

// ShownInfo copied from metrics/completions/metric.go
type ShownInfo struct {
	Source    response.EditorCompletionSource `json:"source"`
	Rank      int                             `json:"rank"`
	Len       int                             `json:"len"`
	NumTokens int                             `json:"num_tokens"`
}

// CompletedInfo copied from metrics/completions/metric.go
type CompletedInfo struct {
	Source        response.EditorCompletionSource `json:"Source"`
	LastShownRank int                             `json:"Rank"`
	LastShownLen  int                             `json:"Len"`

	// num chars/tokens inserted since last shown
	NumCharsInserted  int `json:"NumInserted"`
	NumTokensInserted int `json:"num_tokens_inserted"`

	// num tokens of the full text when first shown
	NumTokensFirstShown int `json:"num_tokens_first_shown"`
}

// SourceBreakdown describes a breakdown of counts by completions source.
type SourceBreakdown map[response.EditorCompletionSource]int

// CompEvent contains a subset of the fields of a kite_status event log that are relevant for analyzing completions
// metrics.
// Note that some fields that are included here are not currently being logged but are present in some historical logs.
type CompEvent struct {
	// Timestamp according to Segment (added after deserialization)
	Timestamp time.Time `json:"timestamp"`

	UserID        interface{} `json:"user_id"`
	ClientVersion string      `json:"client_version"`
	// SentAt as set by the client; note that this is subject to the whims of the client's clock
	SentAt int64 `json:"sent_at"`

	CompletionsRequested           int `json:"completions_requested"`
	CompletionsRequestedExpected   int `json:"completions_requested_expected"`
	CompletionsRequestedUnexpected int `json:"completions_requested_unexpected"`
	CompletionsRequestedError      int `json:"completions_requested_error"`

	CompletionsRequestedRaw           int `json:"completions_requested_raw"`
	CompletionsRequestedExpectedRaw   int `json:"completions_requested_expected_raw"`
	CompletionsRequestedUnexpectedRaw int `json:"completions_requested_unexpected_raw"`
	CompletionsRequestedErrorRaw      int `json:"completions_requested_error_raw"`

	CompletionsCharsInserted int `json:"completions_chars_inserted"`
	CompletionsTriggered     int `json:"completions_triggered"`

	CompletionsAtLeastOneShown    int `json:"completions_at_least_one_shown"`
	CompletionsAtLeastOneNewShown int `json:"completions_at_least_one_new_shown"`

	CompletionsNumShown     int `json:"completions_shown"`
	CompletionsNumSelected  int `json:"completions_num_selected"`
	CompletionsNumCompleted int `json:"completions_num_completed"`
	CompletionsTimeout      int `json:"completions_timeout"`

	CompletionsShownBySource              SourceBreakdown `json:"completions_shown_by_source"`
	CompletionsAtLeastOneShownBySource    SourceBreakdown `json:"completions_at_least_one_shown_by_source"`
	CompletionsAtLeastOneNewShownBySource SourceBreakdown `json:"completions_at_least_one_new_shown_by_source"`

	// These breakdowns were introduced by https://github.com/kiteco/kiteco/pull/8131 and replace
	// CompletionsSelected and CompletionsCompleted. They are represented here as pointers to maps so that we can
	// detect whether the fields were present in the incoming JSON.
	CompletionsSelectedBySource  *SourceBreakdown `json:"completions_selected_by_source"`
	CompletionsSelected2BySource *SourceBreakdown `json:"completions_selected_2_by_source"`
	CompletionsCompletedBySource *SourceBreakdown `json:"completions_completed_by_source"`

	// Note that these are deprecated as of https://github.com/kiteco/kiteco/pull/7671
	CompletionsShown []ShownInfo `json:"completions_shown_sample"`

	// Note that these are deprecated as of https://github.com/kiteco/kiteco/pull/8131
	CompletionsSelected  []CompletedInfo `json:"completions_selected"`
	CompletionsCompleted []CompletedInfo `json:"completions_completed"`
}

// SampleTag implements pipeline.Sample
func (CompEvent) SampleTag() {}

// ComputeBreakdowns calculates the completions selected/completed by source if necessary
// see https://github.com/kiteco/kiteco/pull/8131
func (c *CompEvent) ComputeBreakdowns() {
	if c.CompletionsSelectedBySource != nil {
		// the breakdowns already exist; nothing to be done
		return
	}

	selected := make(SourceBreakdown)
	selected2 := make(SourceBreakdown)
	for _, c := range c.CompletionsSelected {
		selected[c.Source]++
		if c.NumCharsInserted >= 2 {
			selected2[c.Source]++
		}
	}

	completed := make(SourceBreakdown)
	for _, c := range c.CompletionsCompleted {
		completed[c.Source]++
	}

	c.CompletionsSelectedBySource = &selected
	c.CompletionsSelected2BySource = &selected2
	c.CompletionsCompletedBySource = &completed
}
