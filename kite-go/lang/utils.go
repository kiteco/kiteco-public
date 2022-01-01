package lang

import (
	"bufio"
	"hash/fnv"
	"log"
	"math"
	"os"
	"strings"
)

// sum sums all the entries in an input slice
func sum(slice []float64) float64 {
	var sum float64
	for _, value := range slice {
		sum += value
	}
	return sum
}

// hashStrToUint64 hashes an input string to an uint64.
func hashStrToUint64(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// readLines reads all the lines in a file
func readLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// getTokens splits a string by space and period and returns all the tokens
func getTokens(line string) []string {
	var tokens []string
	words := strings.Split(line, " ")
	for _, w := range words {
		tokens = append(tokens, strings.Split(w, ".")...)
	}
	return tokens
}

// getMostLikelyLanguage returns the Language that produces the highest score
// The highest score has to be at least as 5 times larger than as the second
// highest score; otherwise, Unknown is returned.
func getMostLikelyLanguage(scores map[Language]float64) Language {
	score1 := math.Inf(-1)
	score2 := math.Inf(-1)
	predictedLang := Unknown

	// Retrieve the most likely language
	for l, s := range scores {
		if s > score1 {
			score1, score2 = s, score1
			predictedLang = l
		} else if s > score2 {
			score2 = s
		}
	}
	if score1-score2 > math.Log10(5) {
		return predictedLang
	}
	return Unknown
}

// exists returns true if dir exists and prints out error messages
// (as well as return false) if dir doesn't exist
func exists(dir string) bool {
	_, err := os.Stat(dir)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		log.Printf("Error: %s does not exist.\n", dir)
	} else {
		log.Println("Error:", err)
	}
	return false
}
