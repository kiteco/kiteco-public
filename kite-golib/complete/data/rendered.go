package data

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/stringindex"
)

// RenderOptions encapsulates common rendering options
type RenderOptions struct {
	// Disregards the Smart/Regular distinction for prepending stars
	AllCompletionsStarred bool
	// Disables ★ prefix in the hint of "smart" (Pro-only) completions.
	NoSmartStarInHint bool
	// Adds ★ prefix in the display text of "smart" (Pro-only) completions.
	SmartStarInDisplay bool
	// Adds {int}★ prefix in the display text of "smart" (Pro-only) completions.
	SmartStarWithCount bool
	SmartStarCount     int
	// Generate Pro completions, truncated to a single-token (e.g. CTAs, after paywall is exhausted)
	SingleTokenProCompletion bool
	// CTA completions' insert text should not be truncated
	FullProCompletion bool

	// The display hint should be "Pro" for pro completions
	ProHint bool
}

// Documentation encapsulates text documentation
type Documentation struct {
	Text string `json:"text"`
}

// ReferentInfo encapsulates information about the "referent" (value or symbol) of a completion.
// The referent of a completion does not necessarily correspond to the precise value
// the resulting expression might resolve to.
// Instead, it corresponds to what the user might expect to be shown details about when selecting
// the completion: call completions have as their referent the function.
// Currently, the ReferentInfo of a nested completion matches that of the parent completion.
type ReferentInfo struct {
	Docs    Documentation `json:"documentation"`
	WebID   string        `json:"web_id,omitempty"`   // https://kite.com/python/docs/<WebID>
	LocalID string        `json:"local_id,omitempty"` // understood by Kite Local
}

// RCompletion encapsulates a rendered Completion
type RCompletion struct {
	Completion
	Display string `json:"display"`

	ReferentInfo
	Hint     string `json:"hint"`
	Smart    bool   `json:"smart"`
	IsServer bool   `json:"-"`

	// - for debugging, testing & metrics
	Source response.EditorCompletionSource `json:"-"`

	Provider ProviderName `json:"-"`

	Debug   interface{} `json:"-"`
	Metrics interface{} `json:"-"`
}

// NRCompletion encapsulates a rendered Completion for historical purposes:
// nested completions are deprecated.
type NRCompletion struct {
	RCompletion
}

// CompletionIsSmart returns whether the completion should be considered Smart
// It should be called when an RCompl is instantiated
func CompletionIsSmart(c Completion, fromSmartProvider bool, o RenderOptions) bool {
	hasMult, err := c.HasMultiIdents()
	return fromSmartProvider && err == nil && hasMult
}

// AddSmartStar returns true if a star should be added to smart (Kite Pro) RCompletions
func (o RenderOptions) AddSmartStar() bool {
	return !o.NoSmartStarInHint || o.SmartStarInDisplay || o.SmartStarWithCount || o.AllCompletionsStarred
}

// AddSmartStar adds a star if necessary.
// It should be called after the RCompletion is otherwise finalized.
func (c *RCompletion) AddSmartStar(o RenderOptions) {
	if c.Smart || o.AllCompletionsStarred {
		if !o.NoSmartStarInHint {
			c.Hint = joinWithSpace("★", c.Hint)
			if o.SmartStarWithCount {
				c.Hint = strconv.FormatInt(int64(o.SmartStarCount), 10) + c.Hint
			}
		}
		if o.SmartStarInDisplay {
			c.Display = joinWithSpace("★", c.Display)
			if o.SmartStarWithCount {
				c.Display = strconv.FormatInt(int64(o.SmartStarCount), 10) + c.Display
			}
		}
	}
	if c.IsServer {
		if !o.NoSmartStarInHint {
			c.Hint = "★★"
		}
		if o.SmartStarInDisplay {
			c.Display = joinWithSpace("★★", c.Display)
		}
	}
}

// EncodeOffsets encodes offsets according to the given encoding.
func (c *RCompletion) EncodeOffsets(text string, from, to stringindex.OffsetEncoding) error {
	return c.Completion.EncodeOffsets(text, from, to)
}

// EncodeOffsets encodes offsets according to the given encoding.
func (c *NRCompletion) EncodeOffsets(text string, from, to stringindex.OffsetEncoding) error {
	if err := c.RCompletion.EncodeOffsets(text, from, to); err != nil {
		return err
	}
	return nil
}

// DedupeTrailingSpace filters a list of NPCompletions and dedupes the ones
// that only differ by trailing space
func DedupeTrailingSpace(completions []NRCompletion) []NRCompletion {
	var keep []NRCompletion
	seen := make(map[string]bool)
	for _, completion := range completions {
		trimmed := strings.TrimRight(completion.Display, " ")
		if seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		keep = append(keep, completion)
	}
	return keep
}

// -

func joinWithSpace(x, y string) string {
	if x == "" {
		return y
	}
	if y == "" {
		return x
	}
	return fmt.Sprintf("%s %s", x, y)
}
