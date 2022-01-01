package text

import "errors"

// NGrams constructs the n grams (of order n) for the given token stream.
func NGrams(n int, toks []string) ([][]string, error) {
	if n < 1 || len(toks) < n {
		return nil, errors.New("not enough tokens for nGrams")
	}
	var nGrams [][]string
	for i := 0; i+n <= len(toks); i++ {
		var nGram []string
		for j := i; j < i+n; j++ {
			nGram = append(nGram, toks[j])
		}
		nGrams = append(nGrams, nGram)
	}
	return nGrams, nil
}
