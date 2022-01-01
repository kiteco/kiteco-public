package legacy

import (
	"math"
	"path"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/response"
)

const verySmall = 1e-6

func equalFloats(a, b float64) bool {
	return math.Abs(a-b) < verySmall
}

func cmpCompletions(ai, aj Completion) bool {
	// If we're mixing traditional and model-based attribute completions, always rank the
	// traditional ones lower because the assumption is that if the model wasn't trained on them,
	// they're not very significant
	// TODO: perhaps this should be the job of the mixing API, but that would require some rework since
	// currently the API is only responsible for mixing attribute completions
	if ai.Source == response.AttributeModelCompletionSource && aj.Source == response.TraditionalCompletionSource {
		return true
	}

	if ai.Source == response.TraditionalCompletionSource && aj.Source == response.AttributeModelCompletionSource {
		return false
	}

	// sort in descending score order if there is a significant enough difference
	if !equalFloats(ai.Score, aj.Score) {
		return ai.Score > aj.Score
	}

	// break score ties by sorting in ascending alphabetical order
	return ai.Identifier < aj.Identifier
}

func sortByScore(completions []Completion) {
	sort.Slice(completions, func(i, j int) bool {
		return cmpCompletions(completions[i], completions[j])
	})
}

func valueName(val pythontype.Value) string {
	addr := val.Address()
	// if addr.Nil(), we'll end up returning ""
	if addr.Path.Empty() {
		return strings.TrimSuffix(path.Base(addr.File), ".py")
	}
	return addr.Path.Last()
}
