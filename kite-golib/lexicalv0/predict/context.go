package predict

import (
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
)

const (
	naturalWindowPct                         = 0.75
	minNatural                               = 50
	numLinesAfterCursorForContextAfterCursor = 1
)

func buildContextAfter(in Inputs, enc *lexicalv0.FileEncoder, search SearchConfig) []int64 {
	// context after cursor always starts with a sep token
	context := []int64{int64(enc.SepVocabID())}
	if !in.AllowContextAfterCursor {
		return context
	}

	start := in.CursorTokenIdx
	if useCursorTokenForPrefix(in, enc) {
		start++
	}

	if start >= len(in.Tokens) {
		return context
	}

	// find where to start context after cursor
	var newlinePos int
	var newlineCount int
	in.Buffer.Range(in.Tokens[start].Start, func(i int, r rune) bool {
		if r == '\n' {
			newlineCount++
			newlinePos = i
			if newlineCount >= numLinesAfterCursorForContextAfterCursor {
				return false
			}
		}
		return true
	})

	if newlineCount < numLinesAfterCursorForContextAfterCursor {
		return context
	}

	for ; start < len(in.Tokens); start++ {
		if in.Tokens[start].Start >= newlinePos {
			break
		}
	}

	context = append(context, enocdeContextAfter(in.Tokens, enc, start, search.Window)...)
	if len(context) > search.Window {
		context = context[:search.Window]
	}

	return context
}

func useCursorTokenForPrefix(in Inputs, enc *lexicalv0.FileEncoder) bool {
	if len(in.Prefix) > 0 && in.CursorTokenIdx < len(in.Tokens) {
		// TODO: pretty nasty
		// This is mostly for JS, the string associated with the token under the cursor can be very long, this makes it harder
		// to predict the right thing, so to help the model we encode in the context all of the subtokens except
		// the last one, which we use as the prefix.
		subtokens, _ := enc.Lexer.ShouldBPEEncode(in.Tokens[in.CursorTokenIdx])
		return len(subtokens) > 1
	}
	return false
}

func buildContextBeforeAndSetPrefix(in *Inputs, enc *lexicalv0.FileEncoder, search SearchConfig) ([]int64, map[int]bool) {
	natural := encodeContextBefore(in.Tokens, enc, in.CursorTokenIdx, search.Window)

	naturalWindow := int(float64(search.Window) * naturalWindowPct)

	context, numCurated := mergeContexts(newEditorEvents(enc, in.Events), natural, search.Window, naturalWindow, minNatural)

	if useCursorTokenForPrefix(*in, enc) {
		// TODO: pretty nasty
		// This is mostly for JS, the string associated with the token under the cursor can be very long, this makes it harder
		// to predict the right thing, so to help the model we encode in the context all of the subtokens except
		// the last one, which we use as the prefix.
		subtokens, _ := enc.Lexer.ShouldBPEEncode(in.Tokens[in.CursorTokenIdx])
		context = append(context, enc.EncodeSubtokens(subtokens[:len(subtokens)-1])...)
		in.Prefix = enc.Lexer.TrimTerminal(subtokens[len(subtokens)-1])
	}

	context = enc.PrepareBeforeContext(context, search.Window, in.FilePath)

	ctx64 := toInt64(context)

	return ctx64, curatedTokens(enc, ctx64, numCurated)
}

func encodeContextBefore(tokens []lexer.Token, encoder *lexicalv0.FileEncoder, right, window int) []int {
	left := right - window
	if left < 0 {
		left = 0
	}
	context := encoder.EncodeTokens(tokens[left:right])
	if len(context) >= window {
		return context
	}
	for i := left - 1; i >= 0; i-- {
		context = append(encoder.EncodeTokens([]lexer.Token{tokens[i]}), context...)
		if len(context) >= window {
			return context
		}
	}
	return context
}

func enocdeContextAfter(tokens []lexer.Token, encoder *lexicalv0.FileEncoder, left, window int) []int64 {
	right := left + window
	if right > len(tokens) {
		right = len(tokens)
	}
	context := encoder.EncodeTokens(tokens[left:right])
	if len(context) >= window {
		return toInt64(context)
	}
	for i := right; i < len(tokens); i++ {
		context = append(context, encoder.EncodeTokens([]lexer.Token{tokens[i]})...)
		if len(context) >= window {
			return toInt64(context)
		}
	}
	return toInt64(context)
}

type editorEvent struct {
	Encoded []int
}

func newEditorEvents(enc *lexicalv0.FileEncoder, edits EditorEvents) []editorEvent {
	encoded := make([]editorEvent, 0, len(edits))
	for _, edit := range edits {
		encoded = append(encoded, editorEvent{
			Encoded: enc.EncodeTokens(edit.Tokens),
		})
	}
	return encoded
}

// TODO: make this operate on EditorEvent
func mergeContexts(edits []editorEvent, natural []int, window, naturalWindow, minNatural int) ([]int, int) {
	if len(natural) < minNatural || len(edits) == 0 {
		return natural, 0
	}
	if len(natural) > naturalWindow {
		natural = natural[len(natural)-naturalWindow:]
	}

	mergedLength := len(natural)
	i := len(edits)
	for i-1 >= 0 && mergedLength+len(edits[i-1].Encoded) <= window {
		mergedLength += len(edits[i-1].Encoded)
		i--
	}

	merged := make([]int, 0, mergedLength)
	for _, edit := range edits[i:] {
		merged = append(merged, edit.Encoded...)
	}
	merged = append(merged, natural...)

	return merged, len(merged) - len(natural)
}

func curatedContextUsed(curatedTokens map[int]bool, pred []int) bool {
	for _, t := range pred {
		if _, ok := curatedTokens[t]; ok {
			return true
		}
	}
	return false
}

func curatedTokens(enc *lexicalv0.FileEncoder, context []int64, numCuratedTokens int) map[int]bool {
	if numCuratedTokens == 0 {
		return nil
	}
	curated := make(map[int]bool)
	for _, tok := range context[:numCuratedTokens] {
		if !enc.IsLexical(int(tok)) {
			curated[int(tok)] = true
		}
	}
	return curated
}
