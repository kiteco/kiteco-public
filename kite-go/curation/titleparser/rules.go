package titleparser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// emptyTitle checks if the title is empty.
func emptyTitle(parsedTitle *parsedTitle) ([]*TitleViolation, error) {
	var violations []*TitleViolation
	if parsedTitle.parsableTitle == "" {
		violations = append(violations, &TitleViolation{
			Message: "Title should not be empty.",
		})
	}
	return violations, nil
}

// trailingPeriod checks if the title ends with a period.
func trailingPeriod(parsedTitle *parsedTitle) ([]*TitleViolation, error) {
	var violations []*TitleViolation
	title := parsedTitle.parsableTitle
	if strings.HasSuffix(title, ".") {
		violations = append(violations, &TitleViolation{
			Message: "Title should not end with a period.",
		})
	}
	return violations, nil
}

// multipleSentences checks the title consists of only 1 sentence.
func multipleSentences(parsedTitle *parsedTitle) ([]*TitleViolation, error) {
	var violations []*TitleViolation
	parse := parsedTitle.parseTree
	if n := numSentences(parse); n > 1 {
		violations = append(violations, &TitleViolation{
			Message: "Title should consist of only 1 sentence: got " + strconv.Itoa(n) + ".",
		})
	}
	return violations, nil
}

// startWithVerbBase checks if the first word is an infinitive verb.
func startWithVerbBase(parsedTitle *parsedTitle) ([]*TitleViolation, error) {
	var violations []*TitleViolation
	tags := parsedTitle.tags
	if len(tags) > 0 && !isVerb(tags[0]) {
		violations = append(violations, &TitleViolation{
			Message: "Title must start with an infinitive verb: got '" + tags[0].text + "'.",
		})
	}
	return violations, nil
}

// hasApostrophes checks that no apostrophes are used for possession, e.g., thread's.
func hasApostrophes(parsedTitle *parsedTitle) ([]*TitleViolation, error) {
	var violations []*TitleViolation
	tags := parsedTitle.tags
	if t, has := findPossession(tags); has {
		violations = append(violations, &TitleViolation{
			Message: "Avoid apostrophes for the possessive case: \"" + t + "\".",
		})
	}
	return violations, nil
}

// hasContractions checks that no contraction is used, e.g., can't.
func hasContractions(parsedTitle *parsedTitle) ([]*TitleViolation, error) {
	var violations []*TitleViolation
	tags := parsedTitle.tags
	if t, has := findContraction(tags); has {
		violations = append(violations, &TitleViolation{
			Message: "Do not use contraction: got '" + t + "'.",
		})
	}
	return violations, nil
}

// useOneNotAOrAn checks if the title contains the word "one".
func useOneNotAOrAn(parsedTitle *parsedTitle) ([]*TitleViolation, error) {
	var violations []*TitleViolation
	tags := parsedTitle.tags
	for _, t := range tags {
		if t.partofspeech == posCardinalNumber && t.text == "one" {
			violations = append(violations, &TitleViolation{
				Message: "Replace 'one' with a or an.",
			})
		}
	}
	return violations, nil
}

// missingBrackets checks if specs are surrounded by square brackets.
func missingBrackets(parsedTitle *parsedTitle) ([]*TitleViolation, error) {
	// Get the gold standard title
	var violations []*TitleViolation
	parse := parsedTitle.parseTree
	standard := normalizeQuotes(expectedTitle(parse))
	cleanTitle := parsedTitle.cleanTitle

	specs, err := missingSpecs(cleanTitle, standard)
	for _, s := range specs {
		violations = append(violations, &TitleViolation{
			Message: "Possibly missing brackets around '" + s + "'.",
		})
	}
	return violations, err
}

// startWithReturn checks if the sentence does not start with "return".
func startWithReturn(parsedTitle *parsedTitle) ([]*TitleViolation, error) {
	var violations []*TitleViolation
	tags := parsedTitle.tags

	if len(tags) > 0 && strings.ToLower(tags[0].text) == "return" {
		violations = append(violations, &TitleViolation{
			Message: fmt.Sprintf("Do not use '%s' as the main verb.", tags[0].text),
		})
	}
	return violations, nil
}

// hasUseToVB checks that no "use ... to ..." is used in the title.
func hasUseToVB(parsedTitle *parsedTitle) ([]*TitleViolation, error) {
	var violations []*TitleViolation
	parse := parsedTitle.parseTree
	tags := parsedTitle.tags

	vps := verbPhrases(parse)
	for _, vp := range vps {
		if m := useToRegexp.FindStringSubmatch(vp); len(m) > 0 {
			for _, t := range tags {
				if m[1] == t.text && isVerb(t) {
					violations = append(violations, &TitleViolation{
						Message: fmt.Sprintf("'Use ... to %s' is found in the title. Start the title with '%s'.", m[1], m[1]),
					})
				}
			}
		}
	}
	return violations, nil
}

// findSemanticCousins finds semantic cousin if exists
func findSemanticCousins(parsedTitle *parsedTitle, word2vecMap map[string][]float64, verbList strList) ([]*TitleViolation, error) {
	var violations []*TitleViolation
	tags := parsedTitle.tags

	if len(tags) > 0 && isVerb(tags[0]) && !verbList.find(strings.ToLower(tags[0].text)) {
		cousins := similarWords(strings.ToLower(tags[0].text), verbList, word2vecMap)
		if len(cousins) > 0 {
			violations = append(violations, &TitleViolation{
				Message: fmt.Sprintf(
					"Consider replacing '%s' with any of the following word(s): '%s'.",
					tags[0].text, strings.Join(cousins, ", ")),
			})
		}
	}
	return violations, nil
}

// codeArgsNotInTitle checks if there are args in the code that are not mentioned in the title.
func codeArgsNotInTitle(parsedTitle *parsedTitle) ([]*TitleViolation, error) {
	var violations []*TitleViolation
	prelude := parsedTitle.prelude
	code := parsedTitle.code
	title := parsedTitle.parsableTitle
	tags := parsedTitle.tags

	// We only parse args for the functions that a code example is written for, so we check the prelude
	// and see what functions this code example is for.
	exampleOf := importedMethods(prelude)
	exampleOf = append(exampleOf, importedMethods(code)...)

	// Extract args of the imported methods from the code
	args := argsOfFuncs(code, exampleOf)

	// We check whether a function arg is mentioned in the title by checking
	// 1) whether the arg is a substring of the title or
	// 2) whether any of the tokens in the title is a substring of the func arg.
	for _, arg := range args {
		target := removeQuotes(arg.val)
		// do not fire a style violation if it's a list, tuple, or a dict.
		if startsWithBrackets(target) {
			continue
		}
		if ok, _ := regexp.MatchString(target, title); !ok && !substringOverlap(target, tags) {
			if arg.key != "" {
				target = arg.key
				if ok, _ := regexp.MatchString(target, title); !ok && !substringOverlap(target, tags) {
					violations = append(violations, &TitleViolation{
						Message: fmt.Sprintf(
							"'%s=%s' is used as an argument, but neither the keyword nor the value is literally mentioned in the title.", arg.key, arg.val),
					})
				}

			}
		}
	}
	return violations, nil
}

// useShapeAsAdj checks if "shape (int, int)" is in the title.
func useShapeAsAdj(parsedTitle *parsedTitle) ([]*TitleViolation, error) {
	var violations []*TitleViolation
	title := parsedTitle.parsableTitle

	if m := shapeRegexp.FindAllStringSubmatch(title, -1); len(m) > 0 {
		for _, t := range m {
			nums := strings.Split(spaceRegexp.ReplaceAllString(t[2], ""), ",")
			violations = append(violations, &TitleViolation{
				Message: fmt.Sprintf("'%s' is found in title, rephrase the title using '%s'.", t[1], strings.Join(nums, "-by-")),
			})
		}
	}
	return violations, nil
}
