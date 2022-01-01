package tfidf

// sum sums all the entries in an input slice
func sum(slice []float64) float64 {
	var sum float64
	for _, value := range slice {
		sum += value
	}
	return sum
}
