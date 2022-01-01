package lexicalproviders

import (
	"log"
	"strconv"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

const (
	naturalWindowPct = 0.75
	minNatural       = 50
	curatedDistance  = 20
	lineLogSize      = 20
)

var (
	lineLogs = make(map[lang.Language]*lru.Cache)
	logMutex = new(sync.RWMutex)
)

// Reset clears all state
func Reset() {
	logMutex.Lock()
	defer logMutex.Unlock()
	for _, v := range lineLogs {
		v.Purge()
	}
}

// ResetLanguage clears state, which belongs to a given language
func ResetLanguage(language lang.Language) {
	logMutex.Lock()
	defer logMutex.Unlock()
	for k, v := range lineLogs {
		if k == language {
			v.Purge()
		}
	}
}

// Global contains resources that are specific to a file, but are not tied to a specific Buffer (i.e. file contents)
type Global struct {
	UserID       int64
	MachineID    string
	FilePath     string
	Models       *lexicalmodels.Models
	EditorEvents []*component.EditorEvent
	Product      licensing.ProductGetter
}

type pathLine struct {
	language lang.Language
	path     string
	line     int
	tokens   []lexer.Token
}

func newPathLine(raw, path string, line int, langLexer lexer.Lexer) (pathLine, error) {
	lexed, err := langLexer.Lex([]byte(raw))
	if err != nil {
		return pathLine{}, err
	}
	if len(lexed) > 0 && langLexer.IsType(lexer.EOF, lexed[len(lexed)-1]) {
		lexed = lexed[:len(lexed)-1]
	}
	return pathLine{
		language: lang.FromFilename(path),
		path:     path,
		line:     line,
		tokens:   lexed,
	}, nil
}

func (pl pathLine) key() string {
	return pl.path + "#L" + strconv.Itoa(pl.line)
}

// Inputs encapsulates parsed/analyzed inputs to a Provider
type Inputs struct {
	data.SelectedBuffer
	Model           lexicalmodels.ModelBase
	Lexer           lexer.Lexer
	Tokens          []lexer.Token
	PrecededBySpace bool
	LineContext     []lexer.Token
	LastTokenIdx    int

	PredictInputs predict.Inputs

	LangGroup lexicalv0.LangGroup
}

// NewInputs computes inputs from a SelectedBuffer
func NewInputs(ctx kitectx.Context, g Global, b data.SelectedBuffer, allowValueMutation bool) (Inputs, error) {
	originalLang := lang.FromFilename(g.FilePath)
	inputLang := lexicalmodels.LanguageGroupDeprecated(originalLang)

	var model lexicalmodels.ModelBase
	var langGroup lexicalv0.LangGroup
	switch {
	case lexicalv0.MiscLangsGroup.Contains(originalLang):
		model = g.Models.TextMiscGroup
		langGroup = lexicalv0.MiscLangsGroup
	case lexicalv0.WebGroup.Contains(originalLang):
		model = g.Models.TextWebGroup
		langGroup = lexicalv0.WebGroup
	case lexicalv0.JavaPlusPlusGroup.Contains(originalLang):
		model = g.Models.TextJavaGroup
		langGroup = lexicalv0.JavaPlusPlusGroup
	case lexicalv0.CStyleGroup.Contains(originalLang):
		model = g.Models.TextCGroup
		langGroup = lexicalv0.CStyleGroup
	default:
		return Inputs{}, errors.Errorf("unsupported language: %s, %s", inputLang.Name(), g.FilePath)
	}

	langLexer := model.GetLexer()
	tokens, err := lexicalv0.LexSelectedBuffer(b, originalLang, langLexer)
	if err != nil {
		return Inputs{}, err
	}

	cursorContext, err := lexicalv0.FindContext(b, tokens, langLexer)
	if err != nil {
		return Inputs{}, err
	}

	lines, err := encodeEvents(langLexer, g.EditorEvents)
	if err != nil {
		return Inputs{}, err
	}
	updateLog(lines)
	raw, number := extractLine(b.Text(), b.Selection.End)
	line, err := newPathLine(raw, g.FilePath, number, langLexer)
	if err != nil {
		return Inputs{}, err
	}

	curateContextLang := langLexer.Lang()
	if curateContextLang == lang.Text {
		curateContextLang = originalLang
	}

	predictInputs := predict.Inputs{
		FilePath:       g.FilePath,
		Prefix:         cursorContext.Prefix,
		Tokens:         tokens,
		Buffer:         b,
		CursorTokenIdx: cursorContext.LastTokenIdx + 1,
		RandomSeed:     randSeed(b),
		Events:         curateContext(line, curateContextLang),
	}

	inputs := Inputs{
		SelectedBuffer:  b,
		Model:           model,
		Lexer:           langLexer,
		Tokens:          tokens,
		PrecededBySpace: cursorContext.PrecededBySpace,
		LineContext:     cursorContext.LineContext,
		LastTokenIdx:    cursorContext.LastTokenIdx,
		PredictInputs:   predictInputs,
		LangGroup:       langGroup,
	}

	return inputs, nil
}

func encodeEvents(langLexer lexer.Lexer, events []*component.EditorEvent) ([]pathLine, error) {
	var lines []pathLine
	for _, event := range events {
		for _, selection := range event.Selections {
			raw, number := extractLines(event.Text, int(selection.Start), int(selection.End))
			for i := range raw {
				line, err := newPathLine(raw[i], event.Filename, number[i], langLexer)
				if err != nil {
					return nil, err
				}
				lines = append(lines, line)
			}
		}
	}
	return lines, nil
}

func updateLog(lines []pathLine) {
	logMutex.Lock()
	defer logMutex.Unlock()
	for _, line := range lines {
		lineLog, ok := lineLogs[line.language]
		if !ok {
			var err error
			lineLog, err = lru.New(lineLogSize)
			if err != nil {
				log.Fatal(err)
			}
			lineLogs[line.language] = lineLog
		}
		lineLog.Add(line.key(), line)
	}
}

func curateContext(current pathLine, language lang.Language) []predict.EditorEvent {
	logMutex.RLock()
	defer logMutex.RUnlock()

	lineLog, ok := lineLogs[language]
	if !ok {
		return nil
	}

	var curated []predict.EditorEvent
	for _, key := range lineLog.Keys() {
		peeked, ok := lineLog.Peek(key)
		if !ok {
			continue
		}

		line := peeked.(pathLine)
		if useLine(line, current) {
			curated = append(curated, predict.EditorEvent{Tokens: line.tokens})
		}
	}
	return curated
}

func useLine(line, current pathLine) bool {
	if line.language != current.language {
		return false
	}
	if line.path != current.path {
		return true
	}
	if line.line > current.line {
		return true
	}
	if line.line < current.line-curatedDistance {
		return true
	}
	return false
}

func extractLines(text string, start, end int) ([]string, []int) {
	lo := strings.Count(text[:start], "\n")
	hi := strings.Count(text[:end], "\n")
	split := strings.Split(text, "\n")
	var lines []string
	var numbers []int
	for number := lo; number <= hi; number++ {
		// NOTE: In some cases, curating context works better when
		// lines have an automatic semicolon appended to them.
		// It seems complicated to figure out the best treatment
		// for various cases in different languages,
		// so instead we add a new line to the raw code and hope the
		// encoder does something reasonable in most cases.
		lines = append(lines, split[number]+"\n")
		numbers = append(numbers, number)
	}
	return lines, numbers
}

func extractLine(text string, end int) (string, int) {
	number := strings.Count(text[:end], "\n")
	line := strings.Split(text, "\n")[number] + "\n"
	return line, number
}

func randSeed(sb data.SelectedBuffer) int64 {
	// We take the lower 63 bits of the hash of a partial buffer.
	// We use a partial buffer that does not contain the cursor line.
	// This way the seed doesn't change on every keystroke.
	beforeCursor := data.Selection{
		Begin: 0,
		End:   sb.Selection.Begin,
	}
	textBeforeCursor := sb.Buffer.TextAt(beforeCursor)
	cut := strings.LastIndex(textBeforeCursor, "\n") + 1
	partialBuffer := data.NewBuffer(textBeforeCursor[:cut])
	return int64((^uint64(1 << 63)) & partialBuffer.Hash().Hash64())
}
