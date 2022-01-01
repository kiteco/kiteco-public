package titleparser

import (
	"log"
	"regexp"
	"strings"
)

const (
	verb          = `^([^\[\]]+)(\[.*\])*`
	specification = `\[([^\[\]]+)\]`
)

// PosTag represents a part-of-speech tag
type PosTag int

const (
	posNP PosTag = iota
	posVB
	posSPEC
)

type phrase struct {
	tag  PosTag
	text string
}

var (
	parser = newBasicParser()
)

// basicParser is a parser for code example titles.
// It uses regexps to extract the verb phrase and the specs
// out of a title.
type basicParser struct {
	vpParser *regexp.Regexp
	ppParser *regexp.Regexp
}

// newBasicParser returns a basicParser obj, which parses
// the title using regexps designed based on the title template.
func newBasicParser() *basicParser {
	return &basicParser{
		vpParser: regexp.MustCompile(verb),
		ppParser: regexp.MustCompile(specification),
	}
}

// parse parses the title using regexp.
func (bp *basicParser) parse(title string) []*phrase {
	var phrases []*phrase
	m := bp.vpParser.FindStringSubmatch(title)
	if len(m) == 0 {
		log.Printf("0 verb phrases are found for: '%s'\n", title)
		return phrases
	}
	// tokenize the verb phrase
	tokens := strings.Split(m[1], " ")
	// get the verb
	v := strings.TrimSpace(tokens[0])
	// get the noun phrase
	phrases = append(phrases, &phrase{
		tag:  posVB,
		text: v,
	})
	if np := strings.Join(tokens[1:], " "); np != "" {
		np = strings.TrimSpace(np)
		phrases = append(phrases, &phrase{
			tag:  posNP,
			text: np,
		})
	}

	// get the specifications
	n := bp.ppParser.FindAllStringSubmatch(title, -1)
	for _, s := range n {
		phrases = append(phrases, &phrase{
			tag:  posSPEC,
			text: s[1],
		})
	}
	return phrases
}

// RelativeTitle returns the relative title by comparing
// tokens in phrasesA and phrasesB.
func RelativeTitle(titleA, titleB string) string {
	phrasesA := parser.parse(strings.Replace(titleA, "`", "", -1))
	phrasesB := parser.parse(strings.Replace(titleB, "`", "", -1))

	var relativeTokens []string
	for _, p := range phrasesA {
		var flag bool
		var np *phrase
		for _, q := range phrasesB {
			if p.text == q.text {
				flag = true
				break
			}
			if q.tag == posNP {
				np = q
			}
		}
		if flag {
			relativeTokens = append(relativeTokens, p.text)
		} else {
			if p.tag != posNP || np == nil {
				relativeTokens = append(relativeTokens, "<b>"+p.text+"</b>")
			} else {
				tokensA := strings.Split(p.text, " ")
				tokensB := strings.Split(np.text, " ")
				for _, a := range tokensA {
					var matched bool
					for _, b := range tokensB {
						if a == b || skipWord(a) {
							matched = true
							break
						}
					}
					if !matched {
						a = "<b>" + a + "</b>"
					}
					relativeTokens = append(relativeTokens, a)
				}
			}
		}
	}
	return strings.Join(relativeTokens, " ")
}

func skipWord(w string) bool {
	switch w {
	case "a", "an", "the":
		return true
	}
	return false
}
