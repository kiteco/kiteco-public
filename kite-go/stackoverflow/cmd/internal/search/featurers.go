package search

import (
	"errors"
	"log"
	"math"
	"strings"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-golib/text"
	"github.com/kiteco/kiteco/kite-golib/tfidf"
)

var (
	// TextTokenizer is standard tokenizer used in so package for text.
	TextTokenizer = text.NewHTMLTokenizer()
	// TextProcessor is standard text processor used in so package.
	TextProcessor = text.NewProcessor(text.CleanTokens, text.Lower, text.RemoveStopWords, text.Stem)
	// CountProcessor is used for features that need exact word counts (with repetitions included).
	CountProcessor = text.NewProcessor(text.CleanTokens, text.Lower, text.RemoveStopWords, text.Stem)

	// DocTypes are the parts of the SO page that we index separately in
	// terms of tf-idf scores.
	DocTypes = []string{
		"title",
		"body",
		"tags",
		"code",
	}

	// DTSelectors are functions which return the part of the SO page
	// that is part of the given DocType.
	DTSelectors = map[string]func(Document) string{
		DocTypes[0]: titleSelector,
		DocTypes[1]: bodySelector,
		DocTypes[2]: tagSelector,
		DocTypes[3]: codeSelector,
	}

	// DTTokenizers are the tokenizers we use for the different DocTypes.
	DTTokenizers = map[string]text.Tokenizer{
		DocTypes[0]: TextTokenizer,
		DocTypes[1]: TextTokenizer,
		DocTypes[2]: SOTagTokenizer{},
		DocTypes[3]: text.CodeTokenizer{},
	}

	featurerNames = []string{
		"titleTFIDF",
		"bodyTFIDF",
		"tagsTFIDF",
		"codeTFIDF",
		"viewCount",
		"votes",
	}

	allFeaturers = map[string]featurer{
		featurerNames[0]: featurerTFIDF{
			label:     featurerNames[0],
			docType:   DocTypes[0],
			getDoc:    titleSelector,
			tokenizer: TextTokenizer,
			processor: TextProcessor,
		},
		featurerNames[1]: featurerTFIDF{
			label:     featurerNames[1],
			docType:   DocTypes[1],
			getDoc:    bodySelector,
			tokenizer: TextTokenizer,
			processor: TextProcessor,
		},
		featurerNames[2]: featurerTFIDF{
			label:     featurerNames[2],
			docType:   DocTypes[2],
			getDoc:    tagSelector,
			tokenizer: TextTokenizer,
			processor: TextProcessor,
		},
		featurerNames[3]: featurerTFIDF{
			label:     featurerNames[3],
			docType:   DocTypes[3],
			getDoc:    codeSelector,
			tokenizer: text.CodeTokenizer{},
			processor: TextProcessor,
		},
		featurerNames[4]: featurerCounter{
			label: featurerNames[4],
			getCount: func(doc Document) int64 {
				sum := doc.Page.GetQuestion().GetPost().GetViewCount()
				for _, ans := range doc.Page.GetAnswers() {
					sum += ans.GetPost().GetViewCount()
				}
				return sum
			},
		},
		featurerNames[5]: featurerCounter{
			label: featurerNames[5],
			getCount: func(doc Document) int64 {
				sum := doc.Page.GetQuestion().GetPost().GetScore()
				for _, ans := range doc.Page.GetAnswers() {
					sum += ans.GetPost().GetScore()
				}
				return sum
			},
		},
	}

	activeFeatures = featurerNames[:]
)

func codeSelector(doc Document) string {
	codeTokens := text.CodeTokensFromHTML(doc.Page.GetQuestion().GetPost().GetBody())
	for _, ans := range doc.Page.GetAnswers() {
		codeTokens = append(codeTokens, text.CodeTokensFromHTML(ans.GetPost().GetBody())...)
	}
	return strings.Join(codeTokens, " ")
}

func titleSelector(doc Document) string {
	return doc.Page.GetQuestion().GetPost().GetTitle()
}

func qBOdySelector(doc Document) string {
	return doc.Page.GetQuestion().GetPost().GetBody()
}

func tagSelector(doc Document) string {
	tags := doc.Page.GetQuestion().GetPost().GetTags()
	for _, ans := range doc.Page.GetAnswers() {
		tags += " " + ans.GetPost().GetTags()
	}
	return tags
}

func bodySelector(doc Document) string {
	body := doc.Page.GetQuestion().GetPost().GetBody()
	for _, ans := range doc.Page.GetAnswers() {
		body += " " + ans.GetPost().GetBody()
	}
	return body
}

func aBodySelector(doc Document) string {
	var body string
	for _, ans := range doc.Page.GetAnswers() {
		body += " " + ans.GetPost().GetBody()
	}
	return body
}

func aaBodySelector(doc Document) string {
	acceptedAnswerID := doc.Page.GetQuestion().GetPost().GetAcceptedAnswerId()
	for _, ans := range doc.Page.GetAnswers() {
		if ans.GetPost().GetId() == acceptedAnswerID {
			return ans.GetPost().GetBody()
		}
	}
	return ""
}

func naaBodySelector(doc Document) string {
	acceptedAnswerID := doc.Page.GetQuestion().GetPost().GetAcceptedAnswerId()
	var text string
	for _, ans := range doc.Page.GetAnswers() {
		if ans.GetPost().GetId() == acceptedAnswerID {
			continue
		}
		text = text + " " + ans.GetPost().GetBody()
	}
	return text
}

func isBadFloat(f float64) bool {
	return math.IsNaN(f) || math.IsInf(f, 0)
}

// featurer provides generic interface for feature extraction
// only for internal use, clients should use the Featurers interface
type featurer interface {
	Feature(query string, doc Document) float64
	Label() string
}

// featurer based on counts present in the posts, e.g votes or views.
type featurerCounter struct {
	label    string
	getCount func(Document) int64
}

func (fc featurerCounter) Feature(query string, doc Document) float64 {
	feat := float64(fc.getCount(doc))
	if isBadFloat(feat) {
		log.Println(fc.label + ": bad float value")
		return 0
	}
	return feat
}

func (fc featurerCounter) Label() string {
	return fc.label
}

// Featurer based on tf-idf scores
type featurerTFIDF struct {
	idfs      *tfidf.IDFCounter
	label     string
	docType   string
	getDoc    func(Document) string
	tokenizer text.Tokenizer
	processor *text.Processor
}

func rawTermCounts(docTokens []string) map[string]int {
	counts := make(map[string]int)
	for _, t := range docTokens {
		counts[t]++
	}
	return counts
}

func (t featurerTFIDF) Feature(query string, doc Document) float64 {
	queryTokens := t.processor.Apply(t.tokenizer.Tokenize(query))
	docTokens := t.processor.Apply(t.tokenizer.Tokenize(t.getDoc(doc)))
	if len(queryTokens) == 0 || len(docTokens) == 0 {
		return 0.
	}

	var docNorm float64
	dCounts := rawTermCounts(docTokens)
	for tok, count := range dCounts {
		idf := t.idfs.Weight(tok)
		tfidf := float64(count) * idf
		docNorm += tfidf * tfidf
	}
	docNorm = math.Sqrt(docNorm)
	if docNorm < 1e-8 {
		return 0.
	}

	var queryNorm float64
	qCounts := rawTermCounts(queryTokens)
	for tok, count := range qCounts {
		idf := t.idfs.Weight(tok)
		tfidf := float64(count) * idf
		queryNorm += tfidf * tfidf
	}
	queryNorm = math.Sqrt(queryNorm)
	if queryNorm < 1e-8 {
		return 0.
	}

	var tfidfQD float64
	for q, qtf := range qCounts {
		idf := t.idfs.Weight(q)
		dtf, exists := dCounts[q]
		if !exists {
			continue
		}
		tfidfQD += float64(qtf) * idf * float64(dtf) * idf
	}
	tfidfQD /= queryNorm * docNorm

	if isBadFloat(tfidfQD) {
		log.Println(t.label + ": bad float value, query: " + query)
		return 0.
	}
	return tfidfQD
}

func (t featurerTFIDF) Label() string {
	return t.label
}

// Featurers encapsulates an ordered collection of Featurer objects.
type Featurers []featurer

// NewFeaturers returns a new Featurer with an ordered set of Featurer objects.
func NewFeaturers(idfs map[string]*tfidf.IDFCounter) (Featurers, error) {
	var featurers Featurers
	for _, name := range activeFeatures {
		featurer, exists := allFeaturers[name]
		if !exists {
			return nil, errors.New("unsupported featurer name: " + name)
		}
		switch f := featurer.(type) {
		case featurerTFIDF:
			idf, exists := idfs[f.docType]
			if !exists {
				return nil, errors.New("unsupported doc type: " + f.docType)
			}
			f.idfs = idf
			featurers = append(featurers, f)
		case featurerCounter:
			featurers = append(featurers, f)
		default:
			return nil, errors.New("unsupported featurer type: " + f.Label())
		}
	}
	return featurers, nil
}

// Features extracts the feature vector for a given query-document pair
func (fs Featurers) Features(query string, doc Document) []float64 {
	var features []float64
	for _, f := range fs {
		features = append(features, f.Feature(query, doc))
	}
	return features
}

// Labels returns a list of labels for the features the Featurer is extracting
func (fs Featurers) Labels() []string {
	var labels []string
	for _, f := range fs {
		labels = append(labels, f.Label())
	}
	return labels
}

// SOTagTokenizer tokenizes strings of SO tags into tokens
// in which each token is a tag.
type SOTagTokenizer struct{}

// Tokenize returns a list of the SO tags found in the string doc.
func (s SOTagTokenizer) Tokenize(doc string) text.Tokens {
	return SplitTags(doc)
}

// SplitTags splits a string of SO tags into the individual tags.
func SplitTags(str string) []string {
	var (
		tag  string
		tags []string
	)
	for _, ch := range str {
		c := string(ch)
		if c != "" && c != "<" && c != ">" && c != " " {
			tag = tag + c
		}
		if c == ">" && tag != "" {
			tags = append(tags, tag)
			tag = ""
		}
	}
	return tags
}

// TagCount stores the number of times a given tag has appeared
// in the corpus.
type TagCount map[string]int64

// TagClassData stores data neccessary for identifying the (synonym) class
// a tag belongs to, as well as the tags in a given (synonym) class.
type TagClassData struct {
	TagClassIdx map[string]int
	TagClasses  []TagCount
}

// Log encapsulates a query and a list of SO document results for the query
type Log struct {
	Query   string
	Results []Document
}

// Document encapsulates the data associated with a single SO document for feature extraction
type Document struct {
	ID    int64
	URL   string
	Page  *stackoverflow.StackOverflowPage
	Score int
}
