package languagemodel

import "math"

// logSumExp receives a slice of log scores: log(a), log(b), log(c)...
// and returns log(a + b + c....)
func logSumExp(logs []float64) float64 {
	var max float64
	for _, l := range logs {
		if l > max {
			max = l
		}
	}
	var sum float64
	for _, l := range logs {
		sum += math.Exp(l - max)
	}
	return max + math.Log(sum)
}

// sum sums all the entries in an input slice
func sum(slice []float64) float64 {
	var sum float64
	for _, value := range slice {
		sum += value
	}
	return sum
}
