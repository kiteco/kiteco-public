package lexicalv0

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/bpe"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/stretchr/testify/require"
)

func Test_Encoder_Golang(t *testing.T) {
	file1 := `
package main

import (
	"import1"
)

func main() {
	fmt.Println("foo bar")
	fmt.Println(import1.Baz())
	a := func(i, j int) bool {}
}
`

	dir, err := ioutil.TempDir("", "test-vocab")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	vocabPath := fileutil.Join(dir, "vocab.bpe")
	requireVocab(t, file1, vocabPath, lang.Golang)

	encoder, err := NewFileEncoder(vocabPath, NewLangGroup(lang.Golang))
	require.NoError(t, err)

	encoded, err := encoder.Encode([]byte(file1), "file.go")
	require.NoError(t, err)
	require.Equal(t,
		[]string{"SOF", "package", "main$", ";", "import", "(", "STRING", ";", ")", ";", "func", "main$", "(", ")", "{", "fmt$", ".", "Println$", "(", "STRING", ")", ";", "fmt$", ".", "Println$", "(", "i", "m", "p", "o", "r", "t", "1", "$", ".", "B", "a", "z", "$", "(", ")", ")", ";", "a", "$", ":=", "func", "(", "i", "$", ",", "j", "$", "in", "t$", ")", "b", "o", "o", "l", "$", "{", "}", ";", "}", ";", "EOF"},
		encoded,
	)

	encodedIdx, err := encoder.EncodeIdx([]byte(file1), "file.go")
	require.NoError(t, err)
	require.Equal(t,
		[]int{0, 73, 83, 54, 70, 46, 8, 54, 51, 54, 66, 83, 46, 51, 48, 86, 50, 81, 46, 8, 51, 54, 86, 50, 81, 46, 103, 100, 97, 98, 96, 95, 109, 110, 50, 108, 106, 94, 110, 46, 51, 51, 54, 106, 110, 44, 66, 46, 103, 110, 49, 102, 110, 93, 91, 51, 105, 98, 98, 101, 110, 48, 53, 54, 53, 54, 2},
		encodedIdx)
	require.Equal(t, len(encoder.BPE.VocabMap()), 30)
	require.Equal(t, encoder.BPE.VocabMap()["main$"], 2)

	var ids []int
	var lits []string
	decoded := encoder.Decode(encodedIdx)
	for _, dec := range decoded {
		lits = append(lits, dec.Lit)
		ids = append(ids, dec.Token)
	}

	require.Equal(t,
		[]string{"package", "main", ";", "import", "(", "STRING", ";", ")", ";", "func", "main", "(", ")", "{", "fmt", ".", "Println", "(", "STRING", ")", ";", "fmt", ".", "Println", "(", "import1", ".", "Baz", "(", ")", ")", ";", "a", ":=", "func", "(", "i", ",", "j", "int", ")", "bool", "{", "}", ";", "}", ";", "EOF"},
		lits)

	require.Equal(t,
		[]int{78, -1, 57, 75, 49, 9, 57, 54, 57, 71, -1, 49, 54, 51, -1, 53, -1, 49, 9, 54, 57, -1, 53, -1, 49, -1, 53, -1, 49, 54, 54, 57, -1, 47, 71, 49, -1, 52, -1, -1, 54, -1, 51, 56, 57, 56, 57, 1},
		ids)
}

func Test_Encoder_Javascript(t *testing.T) {
	file1 := `
import React from 'react'
import { connect } from 'react-redux'

import '../../assets/setup/plugins.css'

import {
  runningInstallDisable,
  isFullyInstalled,
} from '../../utils/plugins'

import Spinner from '../Spinner'

class SetupPlugins extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      loading: false,
      error: "",
      install: {},
      defaultedInstalls: false,
    }
  }
}
`
	dir, err := ioutil.TempDir("", "test-vocab")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	vocabPath := fileutil.Join(dir, "vocab.bpe")
	requireVocab(t, file1, vocabPath, lang.JavaScript)

	encoder, err := NewFileEncoder(vocabPath, NewLangGroup(lang.JavaScript))
	require.NoError(t, err)

	encoded, err := encoder.Encode([]byte(file1), "file.js")
	require.NoError(t, err)
	require.Equal(t,
		[]string{"SOF", "import", "React$", "from", "'", "r", "eact$", "'", "_automatic_semicolon", "import", "{", "co", "nne", "ct", "$", "}", "from", "'", "r", "eact", "-", "r", "ed", "u", "x", "$", "'", "_automatic_semicolon", "import", "'", "..+", "/+", "..+", "/+", "a", "s", "set", "s+", "/+", "set", "up", "+", "/+", "plugins", ".", "c", "s", "s$", "'", "_automatic_semicolon", "import", "{", "ru", "nn", "ing", "Install", "D", "is", "a", "b", "l", "e$", ",", "is", "F", "u", "ll", "y", "Install", "ed", "$", ",", "}", "from", "'", "..+", "/+", "..+", "/+", "u", "t", "i", "l", "s+", "/+", "plugins", "$", "'", "_automatic_semicolon", "import", "Spinner$", "from", "'", "..+", "/+", "Spinner$", "'", "_automatic_semicolon", "class", "S", "et", "up", "P", "lugins", "$", "extends", "React$", ".", "C", "o", "m", "p", "o", "n", "e", "n", "t", "$", "{", "co", "ns", "t", "ru", "ct", "o", "r$", "(", "props$", ")", "{", "super", "(", "props$", ")", "_automatic_semicolon", "this", ".", "s", "ta", "t", "e$", "=", "{", "l", "o", "a", "d", "ing", "$", ":", "false", ",", "e", "r", "ro", "r$", ":", "\"", "\"", ",", "i", "nstall", "$", ":", "{", "}", ",", "d", "e", "f", "a", "u", "l", "t", "ed", "Install", "s$", ":", "false", ",", "}", "_automatic_semicolon", "}", "}", "_automatic_semicolon"},
		encoded,
	)

	encodedIdx, err := encoder.EncodeIdx([]byte(file1), "file.js")
	require.NoError(t, err)
	require.Equal(t,
		[]int{0, 11, 240, 12, 97, 280, 244, 97, 118, 11, 7, 272, 250, 271, 305, 9, 12, 97, 280, 246, 303, 280, 268, 277, 276, 305, 97, 118, 11, 97, 252, 273, 252, 273, 293, 279, 248, 256, 273, 248, 253, 304, 273, 234, 302, 291, 279, 257, 97, 118, 11, 7, 258, 263, 251, 236, 299, 266, 293, 292, 285, 270, 8, 266, 298, 277, 265, 275, 236, 268, 305, 8, 9, 12, 97, 252, 273, 252, 273, 277, 278, 286, 285, 256, 273, 234, 305, 97, 118, 11, 233, 12, 97, 252, 273, 233, 97, 118, 49, 294, 267, 253, 296, 239, 305, 50, 240, 48, 300, 282, 284, 281, 282, 283, 289, 283, 278, 305, 7, 272, 262, 278, 258, 271, 282, 260, 19, 237, 20, 7, 109, 19, 237, 20, 118, 108, 48, 279, 255, 278, 270, 40, 7, 285, 282, 293, 290, 251, 305, 35, 111, 8, 289, 280, 259, 260, 35, 95, 95, 8, 286, 238, 305, 35, 7, 9, 8, 290, 289, 288, 293, 277, 285, 278, 268, 236, 257, 35, 111, 8, 9, 118, 9, 9, 118},
		encodedIdx)
	require.Equal(t, len(encoder.BPE.VocabMap()), 73)
	require.Equal(t, encoder.BPE.VocabMap()["React$"], 7)

	var ids []int
	var lits []string
	decoded := encoder.Decode(encodedIdx)
	for _, dec := range decoded {
		lits = append(lits, dec.Lit)
		ids = append(ids, dec.Token)
	}
	require.Equal(t,
		[]string{"import", "React", "from", "'", "react", "'", "_automatic_semicolon", "import", "{", "connect", "}", "from", "'", "react-redux", "'", "_automatic_semicolon", "import", "'", "../../assets/setup/plugins.css", "'", "_automatic_semicolon", "import", "{", "runningInstallDisable", ",", "isFullyInstalled", ",", "}", "from", "'", "../../utils/plugins", "'", "_automatic_semicolon", "import", "Spinner", "from", "'", "../Spinner", "'", "_automatic_semicolon", "class", "SetupPlugins", "extends", "React", ".", "Component", "{", "constructor", "(", "props", ")", "{", "super", "(", "props", ")", "_automatic_semicolon", "this", ".", "state", "=", "{", "loading", ":", "false", ",", "error", ":", "\"", "\"", ",", "install", ":", "{", "}", ",", "defaultedInstalls", ":", "false", ",", "}", "_automatic_semicolon", "}", "}", "_automatic_semicolon"},
		lits)

	require.Equal(t,
		[]int{10, -1, 11, 96, -1, 96, 117, 10, 6, -1, 8, 11, 96, -1, 96, 117, 10, 96, -1, 96, 117, 10, 6, -1, 7, -1, 7, 8, 11, 96, -1, 96, 117, 10, -1, 11, 96, -1, 96, 117, 48, -1, 49, -1, 47, -1, 6, -1, 18, -1, 19, 6, 108, 18, -1, 19, 117, 107, 47, -1, 39, 6, -1, 34, 110, 7, -1, 34, 94, 94, 7, -1, 34, 6, 8, 7, -1, 34, 110, 7, 8, 117, 8, 8, 117},
		ids)
}

func Test_Encoder_Python(t *testing.T) {
	file1 := `
import os as operating_system
from sys import args

def main(output):
	with operating_system.open(output, 'w') as fp:
		fp.write(args[0])

if __name__ == '__main__':
	main(args[1])
`

	dir, err := ioutil.TempDir("", "test-vocab")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	vocabPath := fileutil.Join(dir, "vocab.bpe")
	requireVocab(t, file1, vocabPath, lang.Python)

	encoder, err := NewFileEncoder(vocabPath, NewLangGroup(lang.Python))
	require.NoError(t, err)

	encoded, err := encoder.Encode([]byte(file1), "file.js")
	require.NoError(t, err)
	require.Equal(t,
		[]string{"SOF", "import", "o", "s$", "as", "operating_system$", "end_of_statement", "from", "sy", "s$", "import", "args$", "end_of_statement", "def", "main$", "(", "output$", ")", ":", "start_of_block", "with", "operating_system$", ".", "ope", "n", "$", "(", "output$", ",", "string", ")", "as", "fp$", ":", "start_of_block", "fp$", ".", "w", "r", "i", "te", "$", "(", "args$", "[", "integer", "]", ")", "end_of_statement", "end_of_block", "end_of_block", "if", "__", "n", "a", "m", "e", "__", "$", "==", "string", ":", "start_of_block", "main$", "(", "args$", "[", "integer", "]", ")", "end_of_statement", "end_of_block"},
		encoded,
	)

	encodedIdx, err := encoder.EncodeIdx([]byte(file1), "file.js")
	require.NoError(t, err)
	require.Equal(t,
		[]int{0, 3, 262, 247, 10, 224, 220, 5, 246, 247, 3, 236, 220, 34, 235, 7, 231, 8, 23, 221, 33, 224, 4, 242, 263, 271, 7, 231, 9, 190, 8, 10, 243, 23, 221, 243, 4, 256, 260, 265, 245, 271, 7, 236, 79, 88, 80, 8, 220, 222, 222, 22, 254, 263, 269, 264, 268, 254, 271, 58, 190, 23, 221, 235, 7, 236, 79, 88, 80, 8, 220, 222},
		encodedIdx)
	require.Equal(t, len(encoder.BPE.VocabMap()), 48)
	require.Equal(t, encoder.BPE.VocabMap()["main$"], 11)

	var ids []int
	var lits []string
	decoded := encoder.Decode(encodedIdx)
	for _, dec := range decoded {
		lits = append(lits, dec.Lit)
		ids = append(ids, dec.Token)
	}

	require.Equal(t,
		[]string{"import", "os", "as", "operating_system", "end_of_statement", "from", "sys", "import", "args", "end_of_statement", "def", "main", "(", "output", ")", ":", "start_of_block", "with", "operating_system", ".", "open", "(", "output", ",", "string", ")", "as", "fp", ":", "start_of_block", "fp", ".", "write", "(", "args", "[", "integer", "]", ")", "end_of_statement", "end_of_block", "end_of_block", "if", "__name__", "==", "string", ":", "start_of_block", "main", "(", "args", "[", "integer", "]", ")", "end_of_statement", "end_of_block"},
		lits)

	require.Equal(t,
		[]int{2, -1, 9, -1, 219, 4, -1, 2, -1, 219, 33, -1, 6, -1, 7, 22, 220, 32, -1, 3, -1, 6, -1, 8, 189, 7, 9, -1, 22, 220, -1, 3, -1, 6, -1, 78, 87, 79, 7, 219, 221, 221, 21, -1, 57, 189, 22, 220, -1, 6, -1, 78, 87, 79, 7, 219, 221},
		ids)
}

func requireVocab(t *testing.T, contents, vocabPath string, l lang.Language) {
	langLexer, err := NewLexer(l)
	require.NoError(t, err)

	builder := bpe.NewBuilder(false)
	tokens, err := langLexer.Lex([]byte(contents))
	require.NoError(t, err)
	for _, tok := range tokens {
		if subtokens, ok := langLexer.ShouldBPEEncode(tok); ok {
			builder.Add(subtokens)
		}
	}

	builder.Merge(bpe.MergeOptions{})

	f, err := os.Create(vocabPath)
	require.NoError(t, err)
	defer f.Close()

	_, err = builder.WriteTo(f)
	require.NoError(t, err)
}
