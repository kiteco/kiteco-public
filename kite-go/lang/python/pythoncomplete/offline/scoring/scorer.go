package scoring

import (
	"fmt"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/complete/data"

	"github.com/kiteco/kiteco/kite-golib/errors"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/example"
)

// CompletionScore contains a lot of values allowing to score and order completions (but it requires the expected
// completion to be computed so it's an offline scoring
type CompletionScore struct {
	MissingArgs      int  `json:"missing_args"`
	ExtraneousArgs   int  `json:"extraneous_args"`
	IsExactMatch     bool `json:"is_exact_match"`
	PrototypeMatch   bool `json:"prototype_match"`
	PlaceholderCount int  `json:"placeholder_count"`
	ValueMatch       int  `json:"value_match"`
	ValueMismatch    int  `json:"value_mismatch"`
	GlobalScore      int  `json:"global_score"`
}

// ScoreCompletion compares a completion with the expected value to generate a CompletionScore for it
func ScoreCompletion(completion example.Completion, target string) (CompletionScore, error) {
	if strings.Contains(target, "(") {
		return CompletionScore{}, errors.New("Can't score a completion where the target contains another function call (ie '('): \n%s", target)
	}
	cPos, cNamed := processArgs(completion.Identifier)
	tPos, tNamed := processArgs(target)
	return compareArgs(cPos, cNamed, tPos, tNamed)
}

func compareArgs(cPos []string, cNamed map[string]string, tPos []string, tNamed map[string]string) (CompletionScore, error) {
	result := CompletionScore{}
	pArgCount := len(cPos)
	if len(cPos) > len(tPos) {
		result.ExtraneousArgs += len(cPos) - len(tPos)
		pArgCount = len(tPos)
	} else if len(cPos) < len(tPos) {
		result.MissingArgs += len(tPos) - len(cPos)
	}
	for i := 0; i < pArgCount; i++ {
		if isPlaceholder(cPos[i]) {
			result.PlaceholderCount++
		} else if cPos[i] == tPos[i] {
			result.ValueMatch++
		} else {
			result.ValueMismatch++
		}
	}
	for n, cVal := range cNamed {

		targetVal, present := tNamed[n]
		if present {
			if cVal == targetVal {
				result.ValueMatch++
			} else {
				result.ValueMismatch++
			}
		} else {
			result.ExtraneousArgs++
		}
	}
	for n := range tNamed {
		if _, present := cNamed[n]; !present {
			result.MissingArgs++
		}
	}
	result.PrototypeMatch = result.MissingArgs+result.ExtraneousArgs == 0
	result.IsExactMatch = result.PrototypeMatch && (result.ValueMismatch == 0) && (result.PlaceholderCount == 0)
	result.GlobalScore = computeGlobalScore(result)
	return result, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func computeGlobalScore(score CompletionScore) int {
	var result int
	result -= 10 * score.ExtraneousArgs
	result -= 4 * score.MissingArgs
	result += 10 * boolToInt(score.IsExactMatch)
	result -= 2 * score.PlaceholderCount
	result += 5 * boolToInt(score.PrototypeMatch)
	result += 5 * score.ValueMatch
	result -= 5 * score.ValueMismatch
	return result
}

// ScoreDetailString generate a string containing all the details of all non 0 values in this completion score
func ScoreDetailString(score CompletionScore) string {
	var components []string
	if score.ExtraneousArgs > 0 {
		components = append(components, fmt.Sprintf("%d for %d extraneous arg(s)", -10*score.ExtraneousArgs, score.ExtraneousArgs))
	}
	if score.MissingArgs > 0 {
		components = append(components, fmt.Sprintf("%d for %d missing arg(s)", -4*score.MissingArgs, score.MissingArgs))
	}
	if score.PlaceholderCount > 0 {
		components = append(components, fmt.Sprintf("%d for %d placeholder(s)", -2*score.PlaceholderCount, score.PlaceholderCount))
	}
	if score.ValueMatch > 0 {
		components = append(components, fmt.Sprintf("%d for %d value match(es)", 5*score.ValueMatch, score.ValueMatch))
	}
	if score.ValueMismatch > 0 {
		components = append(components, fmt.Sprintf("%d for %d value mismatch(es)", -5*score.ValueMismatch, score.ValueMismatch))
	}
	if score.PrototypeMatch {
		components = append(components, fmt.Sprintf("%d for an exact prototype match", 5))
	}
	if score.IsExactMatch {
		components = append(components, fmt.Sprintf("%d for an exact match", 10))
	}
	return strings.Join(components, "<br/>")
}

func isPlaceholder(s string) bool {
	return strings.HasPrefix(s, data.PlaceholderBeginMark) && strings.HasSuffix(s, data.PlaceholderEndMark)
}

func processArgs(s string) ([]string, map[string]string) {
	names := make(map[string]string)
	pos := make([]string, 0)
	args := getArgs(s)
	for _, arg := range args {
		v, n := getArgComponents(arg)
		if n != "" {
			names[n] = v
		} else {
			pos = append(pos, v)
		}
	}
	return pos, names
}

func getArgs(s string) []string {
	idx := strings.Index(s, ")")
	if idx > 0 {
		s = s[:idx]
	}
	return strings.Split(s, ",")
}

func getArgComponents(s string) (string, string) {
	if strings.Contains(s, "=") {
		idx := strings.Index(s, "=")
		name := strings.TrimSpace(s[:idx])
		value := strings.TrimSpace(s[idx+1:])
		return name, value
	}
	return strings.TrimSpace(s), ""
}
