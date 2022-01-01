package detect

import (
	"github.com/kiteco/kiteco/kite-golib/languagemodel"
)

const (
	// DefaultLanguageDetectorPath is the path to the current default language detector on S3.
	DefaultLanguageDetectorPath = "s3://kite-data/language-detector/2015-11-12_14-29-03-PM/so-title-model.gob"
)

// Detector detects the language(s) associated with a given query or document.
type Detector struct {
	Scorer     *languagemodel.Scorer
	LangSyns   map[string]string
	Thresholds map[string]float64
}

// IsLanguageSynonym returns true in the case that the provided token is a synonym for a language.
// e.g "go" is a language synonym for "golang".
func (d Detector) IsLanguageSynonym(tok string) bool {
	_, found := d.LangSyns[tok]
	return found
}

// Detect detects the language a query is referring to,
// returns the language detected for the query, and true if a language
// tag was found explicitly in the query and false otherwise.
func (d Detector) Detect(tokens []string) map[string]bool {
	// Check for explicit language tags.
	langs := make(map[string]bool)
	for _, tok := range tokens {
		if lang, exists := d.LangSyns[tok]; exists {
			langs[lang] = true
		}
	}
	if len(langs) > 0 {
		// we have found explicit language tags
		return langs
	}
	// No explicit tags, fall back to thresholding posterior for each language.
	posterior := d.Scorer.Posterior(tokens)
	for lang, prob := range posterior {
		if prob > d.Thresholds[lang] {
			// no explicit tags
			langs[lang] = false
		}
	}
	return langs
}
