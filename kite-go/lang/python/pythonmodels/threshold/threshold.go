package threshold

import "fmt"

// Comp stores the score and a boolean indicates if it matches what the user typed.
type Comp struct {
	IsMatched bool
	Score     float32
}

func getTPAndFP(t float32, comps []Comp) (int, int) {
	var tp int
	var fp int
	for _, c := range comps {
		if c.IsMatched {
			if c.Score >= t {
				tp++
			}
		} else {
			if c.Score >= t {
				fp++
			}
		}

	}
	return tp, fp

}

// GetOptimalThreshold searches for the optimal threshold based on true positive rate and false positive rate.
func GetOptimalThreshold(comps []Comp) float32 {
	var neg int
	var pos int
	for _, c := range comps {
		if c.IsMatched {
			pos++
		} else {
			neg++
		}

	}
	if neg == 0 {
		// No negative samples happen in the case of 0 args filetering for partial cases (they are always valid,
		// in this case we want the threshold to be 0)
		return 0
	}
	if pos == 0 {
		panic(fmt.Sprintf("no positive samples"))
	}

	var threshold float32
	var maxDiff float64
	for prob := float32(0); prob < 1.0; prob += 0.001 {
		tp, fp := getTPAndFP(prob, comps)
		tpr := float64(tp) / float64(pos)
		fpr := float64(fp) / float64(neg)

		if diff := tpr - fpr; diff > maxDiff {
			maxDiff = diff
			threshold = prob
		}
	}
	return threshold
}

// Thresholds contains thresholds for different argument
type Thresholds struct {
	ZeroArgs float32
	OneArgs  float32
	TwoArgs  float32
}

func (t Thresholds) String(points0, points1, points2 int) string {
	return fmt.Sprintf("Threshold for 0 args %v (%d data points)\n"+
		"Threshold for 1 args %v (%d data points)\n"+
		"Threshold for 2 args %v (%d data points)\n", t.ZeroArgs, points0, t.OneArgs, points1, t.TwoArgs, points2)
}

// Set group threshold for partial and full calls
type Set struct {
	FullCall    Thresholds `json:"full_call"`
	PartialCall Thresholds `json:"partial_call"`
}

// MTACThreshold is a treshold for MTAC confidence model
type MTACThreshold map[MTACScenario]float32

// MTACScenario defines enum to represent which scenario MTAC is in
type MTACScenario int

// Size is the number of scenarios considered. We don't consider Other for the computation of features so size is only 4
func (s MTACScenario) Size() int {
	return 4
}

const (
	// InCall signifies that MTAC is in an argument of a call.
	InCall MTACScenario = 0
	// InWhile signifies that MTAC is a condition of a while.
	InWhile MTACScenario = 1
	// InIf signifies that MTAC is in a condition of a branch of an if statment or a condition of an if expression.
	InIf MTACScenario = 2
	// InFor signifies that MTAC is in an iterable object of a for loop
	InFor MTACScenario = 3
	// Other signifies all the scopes not considered here
	Other MTACScenario = -1
)
