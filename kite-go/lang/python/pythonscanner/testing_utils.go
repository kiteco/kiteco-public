package pythonscanner

import (
	"bytes"
	"fmt"
	"math/rand"
	"time"
)

const (
	maxRejectSampling = 100
	maxCharsInsert    = 5
	maxCharsEdit      = 5
)

// IncrementalRandom tests the incremental parser by randomly
// altering lines of code and checking that the incremental results are consistent
// with the results of a full lex.
// returns num fails, num relexed, num skipped, num ok
// TODO(juan): test several modification in a row, not just one offs.
// TODO(juan): insert at random cursor points?
func IncrementalRandom(src []byte, iters int, rgen *rand.Rand, debugOut bool) (int, int, int, int) {
	words, _ := Lex(src, Options{
		ScanComments: true,
		ScanNewLines: true,
	})

	var skipped, failed, relexed int
	for i := 0; i < iters; i++ {

		var word Word
		var replacement []byte
		var found bool
		for j := 0; j < maxRejectSampling; j++ {
			// pick a random (non forbidden) word
			idx := rgen.Intn(len(words))
			word = words[idx]
			if hasSpecialWords(word) {
				continue
			}

			replacement, found = modifyWord(src, word, rgen)
			if found {
				break
			}
		}

		if !found {
			skipped++
			continue
		}

		update := &Increment{
			Begin:       word.Begin,
			End:         word.End,
			Replacement: replacement,
		}

		newSrc := bytes.Join([][]byte{
			src[:word.Begin],
			replacement,
			src[word.End:],
		}, nil)

		incr := NewIncrementalFromBuffer([]byte(src), Options{
			ScanComments: true,
			ScanNewLines: true,
		})
		if incr.Update(update) != nil {
			relexed++
			continue
		}

		if updateFailed(string(src), string(newSrc), incr, debugOut) {
			if debugOut {
				fmt.Println("OriginalLine:", text(src, word))
				fmt.Println("NewLine:", text(newSrc, word))
			}
			failed++
		}
	}

	return failed, relexed, skipped, iters - failed - relexed - skipped
}

func text(src []byte, word Word) string {
	start, end := -1, -1
	for i, ch := range src {
		if ch == byte('\n') || ch == byte('\r') {
			if i < int(word.Begin) {
				start = i
			}
			if i >= int(word.End) {
				end = i
			}
		}
		if start != -1 && end != -1 {
			break
		}
	}
	if start == -1 {
		if end == -1 {
			return string(src[:word.End])
		}
		return string(src[:end])
	}
	if end == -1 {
		return string(src[start:])
	}
	return string(src[start:end])
}

func modifyWord(src []byte, word Word, rgen *rand.Rand) ([]byte, bool) {
	orig := src[word.Begin:word.End]
	if len(orig) == 0 {
		return nil, false
	}
	replacement := make([]byte, len(orig))
	copy(replacement, orig)

	switch word.Token {
	case Ident:
		action := rgen.Intn(3)
		switch action {
		case 0:
			//delete
			return nil, true
		case 1:
			// insert
			nInsert := rgen.Intn(maxCharsInsert)
			for i := 0; i < nInsert; i++ {
				pos := rgen.Intn(len(replacement))
				replacement = bytes.Join([][]byte{
					replacement[:pos],
					[]byte{randomLetter(rgen)},
					replacement[pos:],
				}, nil)
			}
			return replacement, true
		default:
			//edit
			nEdit := rgen.Intn(maxCharsEdit)
			for i := 0; i < nEdit; i++ {
				pos := rgen.Intn(len(replacement))
				replacement[pos] = randomLetter(rgen)
			}
			return replacement, true
		}

	case Int, Float, Long:
		action := rgen.Intn(3)
		switch action {
		case 0:
			//delete
			return nil, true
		case 1:
			// insert
			nInsert := rgen.Intn(maxCharsInsert)
			for i := 0; i < nInsert; i++ {
				pos := rgen.Intn(len(replacement))
				replacement = bytes.Join([][]byte{
					replacement[:pos],
					[]byte{randomDigit(rgen)},
					replacement[pos:],
				}, nil)
			}
			return replacement, true
		default:
			//edit
			nEdit := rgen.Intn(maxCharsEdit)
			for i := 0; i < nEdit; i++ {
				pos := rgen.Intn(len(replacement))
				replacement[pos] = randomDigit(rgen)
			}
			return replacement, true
		}
	}
	return nil, false
}

func randomDigit(rgen *rand.Rand) byte {
	return byte('0' + rgen.Intn('9'-'0'+1))
}

func randomLetter(rgen *rand.Rand) byte {
	return byte('a' + rgen.Intn('z'-'a'+1))
}

func updateFailed(src, newSrc string, incr *Incremental, debugOut bool) bool {
	// make sure underlying buffer is correct
	if len(newSrc) != len(incr.buf) {
		return true
	}

	for i := range newSrc {
		if newSrc[i] != incr.buf[i] {
			return true
		}
	}

	// make sure words are correct
	expectedWords, _ := Lex([]byte(newSrc), Options{
		ScanComments:      true,
		ScanNewLines:      true,
		OneBasedPositions: false,
	})

	var wroteHeader bool

	if len(expectedWords) != len(incr.words) {
		if debugOut {
			wroteHeader = true
			fmt.Println("Fail", time.Now())
			end := len(expectedWords)
			if len(incr.words) < end {
				end = len(incr.words)
			}
			for i := 0; i < end; i++ {
				if expectedWords[i] != incr.words[i] {
					fmt.Println("expected:", expectedWords[i].Begin, expectedWords[i].End, expectedWords[i].Literal, expectedWords[i].Token)
					fmt.Println("actual:", incr.words[i].Begin, incr.words[i].End, incr.words[i].Literal, incr.words[i].Token)
				}
			}
		}
		return true
	}

	for i := range expectedWords {
		if expectedWords[i] != incr.words[i] {
			if debugOut {
				if !wroteHeader {
					fmt.Println("Fail", time.Now())
					wroteHeader = true
				}
				if expectedWords[i] != incr.words[i] {
					fmt.Println("expected:", expectedWords[i].Begin, expectedWords[i].End, expectedWords[i].Literal, expectedWords[i].Token)
					fmt.Println("actual:", incr.words[i].Begin, incr.words[i].End, incr.words[i].Literal, incr.words[i].Token)
				}
			}
			return true
		}
	}

	// make sure lines are correct
	expectedLines := lines([]byte(newSrc))
	if len(incr.lines) != len(expectedLines) {
		return true
	}

	for i := range expectedLines {
		if expectedLines[i] != incr.lines[i] {
			return true
		}
	}

	return false
}
