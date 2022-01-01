package pythonkeyword

import (
	"fmt"
	"go/token"
	"time"
	"unicode"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const (
	// ModelLookback describes the number of tokens before the cursor to use as inputs for inference.
	ModelLookback = 5
)

// ModelInputs contains the necessary information used to construct Features.
type ModelInputs struct {
	Buffer    []byte
	Cursor    int64
	AST       pythonast.Node
	Words     []pythonscanner.Word
	ParentMap map[pythonast.Node]pythonast.Node
}

// Features describes the features used for training/inference in the model
// Note that the fields here are presented in the same order in which they're concatenated into the feed tensor
type Features struct {
	// The type of the sibling node to the left of the current statement, 0 if none
	LastSibling int
	// The type of the parent node of the current statement
	ParentNode int

	// the first non-whitespace token of the current statement
	FirstToken pythonscanner.Token

	// The indent level of the current statement relative to the statement on the previous line
	// 0 if same, 1 if shallower, 2 if deeper
	RelIndent int

	// The first char of currently typed token, used as a 1-letter prefix information
	// Only keep lower case letter, -1 when there's no prefix, 0 when it's an invalid prefix, [1-26] for any valid prefix
	// the -1 corresponds to nothing being sent to the model

	FirstChar int64

	// The n tokens preceding the current one, left-padded with 0
	Previous []pythonscanner.Token

	// The keywords that have been seen in the document.
	// That usually gives a lot of information about the type of file currently written (contains assert or classes)
	// Array of 30 values matching keywords category (0-indexed so index 0 corresponds to cat 1)
	// It could be interesting to have a count instead of just a boolean
	//TODO rename as keywordInDoc (as it spans over the whole doc and not only the previous tokens
	PreviousKeywords []int64

	CodeSnippet string `json:"-"`
}

// Vector returns an slice that can be used directly as an input to the keyword model.
func (f Features) Vector() []int64 {
	feats := []int64{
		int64(f.LastSibling),
		int64(f.ParentNode),
		int64(f.FirstToken),
		int64(f.RelIndent),
	}

	for _, p := range f.Previous {
		feats = append(feats, int64(p))
	}
	feats = append(feats, int64(f.FirstChar))
	feats = append(feats, f.PreviousKeywords...)

	return feats
}

// NewFeatures creates a set of features from a set of ModelInputs.
// Most of the file went through a big refactor to be word based instead of being statement based.
// Being stmt based was problematic when the cursor was on a new line.
// In this case the previous line was considered as the current statement
// (or in some case, the cur stmt couldn't be found and the model wasn't running).
func NewFeatures(ctx kitectx.Context, inputs ModelInputs, lookback int) (Features, error) {
	ctx.CheckAbort()

	start := time.Now()
	defer func() {
		newFeaturesDuration.RecordDuration(time.Since(start))
	}()

	curWord, curWordIdx, err := curWord(inputs.Words, inputs.Cursor)
	if err != nil {
		return Features{}, fmt.Errorf("could not find current word: %v", err)
	}

	relIndent, err := relIndent(inputs.Words, curWordIdx)
	if err != nil {
		return Features{}, fmt.Errorf("could not get relative indent: %v", err)
	}

	lastSibling, err := lastSibling(ctx, inputs.Words, curWordIdx, inputs.AST)
	if err != nil {
		return Features{}, fmt.Errorf("could not get last node: %v", err)
	}

	parentNode, err := parentNode(ctx, inputs.Words, curWordIdx, inputs.AST)
	if err != nil {
		return Features{}, fmt.Errorf("could not find parent node: %v", err)
	}

	fToken, err := firstToken(curWordIdx, inputs.Words)
	if err != nil {
		return Features{}, fmt.Errorf("could not get first token: %v", err)
	}

	previousKeywords := previousKeywords(inputs.Words)

	firstChar := getFirstChar(inputs.Buffer, curWord, inputs.Cursor)

	codeSnippet := extractCodeSnippet(inputs.Buffer, inputs.Cursor)

	return Features{
		Previous:         prevTokens(inputs.Words, curWordIdx, lookback),
		LastSibling:      lastSibling,
		ParentNode:       parentNode,
		FirstToken:       fToken,
		FirstChar:        firstChar,
		RelIndent:        relIndent,
		PreviousKeywords: previousKeywords,
		CodeSnippet:      codeSnippet,
	}, nil
}

func previousKeywords(words []pythonscanner.Word) []int64 {
	result := make([]int64, NumKeywords())
	for _, w := range words {
		if w.Token.IsKeyword() {
			cat := KeywordTokenToCat(w.Token)
			if cat > 0 {
				result[cat-1] = 1
			}
		}
	}
	return result
}

func isIndentDedent(tok pythonscanner.Token) bool {
	return tok == pythonscanner.Indent || tok == pythonscanner.Dedent
}

func isIndentDedentNewLine(tok pythonscanner.Token) bool {
	return tok == pythonscanner.Indent || tok == pythonscanner.Dedent || tok == pythonscanner.NewLine
}

func isLineBegin(words []pythonscanner.Word, curWord int) bool {
	if len(words) == 0 || curWord == 0 {
		return true
	}

	if isIndentDedentNewLine(words[curWord].Token) {
		return true
	}

	previousToken := words[curWord-1].Token
	if isIndentDedentNewLine(previousToken) {
		return true
	}
	return false
}

func curWord(words []pythonscanner.Word, cursor int64) (pythonscanner.Word, int, error) {
	var w pythonscanner.Word
	var wordIdx int
	for idx, word := range words {
		if int64(word.Begin) <= cursor && int64(word.End) >= cursor {
			w = word
			wordIdx = idx
			break
		}
	}

	if w == (pythonscanner.Word{}) {
		return pythonscanner.Word{}, -1, fmt.Errorf("could not find current word")
	}

	return w, wordIdx, nil
}

func extractCodeSnippet(buffer []byte, cursor int64) string {
	begin := 0
	end := cursor + 50
	if end > int64(len(buffer)) {
		end = int64(len(buffer))
	}
	return string(buffer[begin:cursor]) + "#$#" + string(buffer[cursor:end])
}

func getFirstChar(buffer []byte, curWord pythonscanner.Word, cursor int64) int64 {
	prefix := string(buffer[curWord.Begin:cursor])
	// If no prefix, return -1 directly
	if len(prefix) < 1 {
		return -1
	}

	firstChar := []rune(prefix)[0]
	if !unicode.IsLetter(firstChar) || !unicode.IsLower(firstChar) {
		return 0
	}
	result := int64(firstChar-'a') + 1
	if result < 1 || result > 26 {
		return 0 // That's a non ascii letter, we don't want it as a prefix
	}
	return result
}

func prevTokens(words []pythonscanner.Word, wordIdx int, count int) []pythonscanner.Token {
	prev := make([]pythonscanner.Token, count)
	wordIdx--
	if wordIdx < 0 {
		// We are at the beginning of the doc, no previous tokens
		return prev
	}
	if words[wordIdx].Token == pythonscanner.NewLine {
		// We skip any final NewLine as there's already a feature telling if we are at the beginning of a line
		wordIdx--
	}

	currentIdx := count - 1
	for ; wordIdx >= 0 && currentIdx >= 0; wordIdx-- {
		word := words[wordIdx]
		// skip Indent tokens
		if word.Token == pythonscanner.Indent {
			continue
		}
		if word.Token == pythonscanner.Dedent {
			continue
		}
		prev[currentIdx] = words[wordIdx].Token
		currentIdx--
	}

	return prev
}

func skipIndentNewLineForward(words []pythonscanner.Word, wordIdx int) int {
	for wordIdx < len(words) && isIndentDedentNewLine(words[wordIdx].Token) {
		wordIdx++
	}
	return wordIdx

}

// getLineBegin goes back to the previous NewLine token
// and count the number of indent/dedent between the initial position and the NewLine token
// It returns a positive number if the current line is indented compared to the previous one
func getLineBegin(words []pythonscanner.Word, wordIdx int) (int, int) {
	var acc int
	for ; wordIdx > 0 && words[wordIdx].Token != pythonscanner.NewLine; wordIdx-- {
		tok := words[wordIdx].Token
		if tok == pythonscanner.Indent {
			acc++
		} else if tok == pythonscanner.Dedent {
			acc--
		}
	}
	return wordIdx, acc
}

// getPreviousLineBegin goes back to the beginning of the previous line
// It also returns the indentation of the previous line
// If the cursor is currently on the first line of the buffer, return 0, 0
func getPreviousLineBegin(words []pythonscanner.Word, wordIdx int) (int, int) {
	wordIdx, _ = getLineBegin(words, wordIdx)
	if wordIdx <= 0 {
		return 0, 0
	}
	return getLineBegin(words, wordIdx-1)
}

func getStatementCategory(ctx kitectx.Context, begin token.Pos, end token.Pos, ast pythonast.Node) int {
	var stmt pythonast.Node
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}
		ctx.CheckAbort()

		if n.Begin() <= begin && n.End() >= end {
			stmt = n
			return true
		}
		return false

	})
	if !pythonast.IsNil(stmt) {
		return NodeToCat(stmt)
	}
	return -1
}

// lastSibling looks for a previous line with the same indentation in the same scope
// If such a line exists, it returns the category of the node corresponding to this line statement (NodeToCat result)
// If no such line exists, it returns 0
// TODO(Moe): Not completely sure how comments as a multiline string are considered.
func lastSibling(ctx kitectx.Context, words []pythonscanner.Word, wordIdx int, ast pythonast.Node) (int, error) {
	ctx.CheckAbort()
	wordIdx = skipIndentNewLineForward(words, wordIdx)
	wordIdx, indentDelta := getLineBegin(words, wordIdx)

	if indentDelta > 0 {
		// If we just got indented, that means we don't have a previous sibling
		return 0, nil
	}

	// Then we look for the previous line with the exact same indentation than us
	var delta int
	for indentDelta < 0 && wordIdx > 0 {
		wordIdx, delta = getPreviousLineBegin(words, wordIdx)
		indentDelta += delta
	}
	if wordIdx <= 0 {
		// We are at the beginning of the file, so we have no previous siblings
		return 0, nil
	}
	// indentDelta is 0, that means the previous line is our previous sibling
	// wordIdx is on the current NewLine token, so wordIdx - 1 is the last word of the previous line
	end := words[wordIdx-1].End
	wordIdx, delta = getPreviousLineBegin(words, wordIdx)
	// Now wordIdx is on the first token of the previous line (so our previous sibling)

	// move to the first real word as the current one is a newLine and it might be followed by indents
	wordIdx = skipIndentNewLineForward(words, wordIdx)

	// so we are now on the first token of our previous sibling
	begin := words[wordIdx].Begin // That's where our previous sibling starts

	return getStatementCategory(ctx, begin, end, ast), nil
}

func getParentStatementCategory(ctx kitectx.Context, begin token.Pos, ast pythonast.Node) int {
	var parent pythonast.Node
	var maybeParent pythonast.Node

	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}
		ctx.CheckAbort()
		if n.Begin() == begin && n.End() > begin {
			parent = n
			return false
		}
		if n.Begin() <= begin && n.End() > begin {
			maybeParent = n
			return true
		}
		return false

	})

	if pythonast.IsNil(parent) {
		parent = maybeParent
	}

	return NodeToCat(parent)
}

// parentNode looks for a previous line with the one level less indentation
// If such a line exists, it returns the category of the node corresponding to this line statement
// If no such line exists, it returns 0
func parentNode(ctx kitectx.Context, words []pythonscanner.Word, wordIdx int, ast pythonast.Node) (int, error) {
	ctx.CheckAbort()

	wordIdx, _ = getLineBegin(words, wordIdx)
	if wordIdx <= 0 {
		// We are in the first line of the page parent is the module
		return NodeToCat(ast), nil
	}
	// wordIdx point to the NewLine token of the current line
	// We first move the cursor after the indents
	wordIdx = skipIndentNewLineForward(words, wordIdx)

	//Then count them
	wordIdx, indentDelta := getLineBegin(words, wordIdx)

	for delta := 0; indentDelta <= 0 && wordIdx > 0; {
		wordIdx, delta = getPreviousLineBegin(words, wordIdx)
		indentDelta += delta
	}

	// the previous line is our parent line
	// Let's place wordIdx at the beginning of the previous line
	wordIdx, _ = getPreviousLineBegin(words, wordIdx)
	wordIdx = skipIndentNewLineForward(words, wordIdx)

	if indentDelta > 0 {
		begin := words[wordIdx].Begin
		nodeCat := getParentStatementCategory(ctx, begin, ast)
		if nodeCat == -1 {
			return NodeToCat(ast), fmt.Errorf("impossible to find parent statement, defaulting to module")
		}
		return nodeCat, nil
	}

	// We have no indentation, our parent is the module
	return NodeToCat(ast), nil

}

func relIndent(words []pythonscanner.Word, curWordIdx int) (int, error) {

	curWordIdx = skipIndentNewLineForward(words, curWordIdx)

	_, indent := getLineBegin(words, curWordIdx)
	if indent > 0 {
		return 2, nil
	}
	if indent < 0 {
		return 1, nil
	}
	return 0, nil
}

func firstToken(wordIdx int, words []pythonscanner.Word) (pythonscanner.Token, error) {
	isLineBegin := isLineBegin(words, wordIdx)
	if isLineBegin {
		return pythonscanner.NewLine, nil
	}
	wordIdx, _ = getLineBegin(words, wordIdx)
	// getLineBegin places wordIdx on the newLine token, we move one token forward to get the actual first token of the line
	wordIdx++
	for wordIdx < len(words) && isIndentDedent(words[wordIdx].Token) {
		// We skip all the indent/dedent
		wordIdx++
	}
	if wordIdx == len(words) || words[wordIdx].Token == pythonscanner.EOF {
		// If we are at the end of the file, we use the NewLine token to tell the model there's nothing yet on the line
		return pythonscanner.NewLine, nil
	}
	return words[wordIdx].Token, nil
}
