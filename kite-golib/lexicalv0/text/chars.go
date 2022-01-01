package text

import "unicode"

type cc int

const (
	unk cc = iota
	letter
	number
	hspace
	vspace
	punct
	symb
)

func charCategory(r rune) cc {
	switch {
	case unicode.IsLetter(r):
		return letter
	case unicode.IsNumber(r) || unicode.IsDigit(r):
		return number
	case r == '\t' || r == ' ' || r == '\u00A0':
		return hspace
	case unicode.IsSpace(r):
		return vspace
	case unicode.IsPunct(r):
		return punct
	case unicode.IsSymbol(r):
		return symb
	default:
		return unk
	}
}

func isPunct(c cc) bool {
	return c == punct
}

func isSpace(c cc) bool {
	return c == hspace || c == vspace
}

func isLetterOrNumber(c cc) bool {
	return c == letter || c == number
}

func isHSpaceWord(s string) bool {
	for _, r := range s {
		if charCategory(r) != hspace {
			return false
		}
	}
	return true
}
