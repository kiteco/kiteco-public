package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	pathSmallFile  = "local-pipelines/lexical/train/cmds/datagen/testdata/small.js"
	pathMediumFile = "local-pipelines/lexical/train/cmds/datagen/testdata/medium.js"
	pathLargeFile  = "local-pipelines/lexical/train/cmds/datagen/testdata/large.js"
	pathTinyFile   = "local-pipelines/lexical/train/cmds/datagen/testdata/tiny.js"
	vocabPath      = "local-pipelines/lexical/train/cmds/datagen/testdata/ident-vocab-500-entries.bpe"

	testFilePaths = []string{}
)

func init() {
	localDataRoot := os.Getenv("GOPATH")
	if localDataRoot != "" {
		localDataRoot = filepath.Join(localDataRoot, "src/github.com/kiteco/kiteco")
	}
	pathTinyFile = filepath.Join(localDataRoot, pathTinyFile)
	pathSmallFile = filepath.Join(localDataRoot, pathSmallFile)
	pathMediumFile = filepath.Join(localDataRoot, pathMediumFile)
	pathLargeFile = filepath.Join(localDataRoot, pathLargeFile)
	testFilePaths = []string{
		pathSmallFile,
		pathMediumFile,
		pathTinyFile,
	}

	vocabPath = filepath.Join(localDataRoot, vocabPath)
}

func assertReadFile(t *testing.T, path string) []byte {
	content, err := ioutil.ReadFile(path)
	assert.NoError(t, err)
	return content
}

func extractorArgs(t *testing.T, l lang.Language) (*sync.Mutex, int, *lexicalv0.FileEncoder, map[string][]int) {
	m := &sync.Mutex{}
	contextSize := 512
	enc, err := lexicalv0.NewFileEncoder(vocabPath, lexicalv0.NewLangGroup(l))
	assert.NoError(t, err)
	concatMap := make(map[string][]int)
	return m, contextSize, enc, concatMap
}

func Test_BasicSampleGeneration(t *testing.T) {
	f := assertReadFile(t, pathMediumFile)
	key := fmt.Sprintf("localfile:%s", pathMediumFile)
	m, cSize, enc, concatMap := extractorArgs(t, lang.JavaScript)
	samples, err := extractSample(key, f, m, cSize, enc, concatMap, 0)
	assert.NoError(t, err)
	assert.NotEmpty(t, samples)
}

func Test_MinifiedNoSample(t *testing.T) {
	f := assertReadFile(t, pathMediumFile)
	m, cSize, enc, concatMap := extractorArgs(t, lang.JavaScript)
	samples, err := extractSample("localfile:some_file.min.js", f, m, cSize, enc, concatMap, 0)
	assert.NoError(t, err)
	assert.Empty(t, samples)
}

func Test_TooShortNoSample(t *testing.T) {
	path := pathSmallFile
	f := assertReadFile(t, path)
	key := fmt.Sprintf("localfile:%s", path)
	m, cSize, enc, concatMap := extractorArgs(t, lang.JavaScript)
	samples, err := extractSample(key, f, m, cSize, enc, concatMap, 0)
	assert.NoError(t, err)
	assert.Empty(t, samples)
}

func Test_ConcatMapOneExt(t *testing.T) {
	path := pathSmallFile
	f := assertReadFile(t, path)
	key := fmt.Sprintf("localfile:%s", path)
	m, cSize, enc, concatMap := extractorArgs(t, lang.JavaScript)
	for i := 0; i < 6; i++ {
		samples, err := extractSample(key, f, m, cSize, enc, concatMap, 0)
		assert.NoError(t, err)
		assert.Empty(t, samples)
		assert.Len(t, concatMap, 1, "The length of concat map should be exactly 1")

	}
	samples, err := extractSample(key, f, m, cSize, enc, concatMap, 0)
	assert.NoError(t, err)
	assert.NotEmpty(t, samples)
	assert.Len(t, concatMap, 1, "The length of concat map should be exactly 1")
	assert.Nil(t, concatMap[".js"])

}

func Test_ConcatMultipleExt(t *testing.T) {
	path := pathSmallFile
	f := assertReadFile(t, path)
	m, cSize, enc, concatMap := extractorArgs(t, lang.JavaScript)
	samples, err := extractSample("localfile:file.js", f, m, cSize, enc, concatMap, 0)
	assert.NoError(t, err)
	assert.Empty(t, samples)
	assert.Len(t, concatMap, 1, "The length of concat map should be exactly 1")
	samples, err = extractSample("localfile:file.jsx", f, m, cSize, enc, concatMap, 0)
	assert.NoError(t, err)
	assert.Empty(t, samples)
	assert.Len(t, concatMap, 2, "The length of concat map should be exactly 2")
	samples, err = extractSample("localfile:file.vue", f, m, cSize, enc, concatMap, 0)
	assert.NoError(t, err)
	assert.Empty(t, samples)
	assert.Len(t, concatMap, 3, "The length of concat map should be exactly 3")
	for i := 0; i < 6; i++ {
		samples, err = extractSample("localfile:file.js", f, m, cSize, enc, concatMap, 0)
	}
	assert.NoError(t, err)
	assert.NotEmpty(t, samples)
	assert.Len(t, concatMap, 3, "The length of concat map should be exactly 3")
	assert.Nil(t, concatMap[".js"])
}

func Test_TooShortNoConcatSmallContext(t *testing.T) {
	path := pathSmallFile
	f := assertReadFile(t, path)
	key := fmt.Sprintf("localfile:%s", path)
	m, _, enc, concatMap := extractorArgs(t, lang.JavaScript)
	smallContext := 400
	samples, err := extractSample(key, f, m, smallContext, enc, concatMap, 512)
	assert.NoError(t, err)
	assert.Empty(t, samples)
	assert.Empty(t, concatMap)
}

func Test_SamplesForSmallFileWithSmallContext(t *testing.T) {
	path := pathSmallFile
	f := assertReadFile(t, path)
	key := fmt.Sprintf("localfile:%s", path)
	m, _, enc, concatMap := extractorArgs(t, lang.JavaScript)
	smallContext := 400
	samples, err := extractSample(key, f, m, smallContext, enc, concatMap, 0)
	assert.NoError(t, err)
	assert.Empty(t, samples)
	smallerContext := 200
	samples, err = extractSample(key, f, m, smallerContext, enc, concatMap, 0)
	assert.NoError(t, err)
	assert.NotEmpty(t, samples)
}

func Test_MediumSmallContext(t *testing.T) {
	path := pathMediumFile
	f := assertReadFile(t, path)
	key := fmt.Sprintf("localfile:%s", path)
	m, _, enc, concatMap := extractorArgs(t, lang.JavaScript)
	smallContext := 400
	samples, err := extractSample(key, f, m, smallContext, enc, concatMap, 0)
	assert.NoError(t, err)
	assert.NotEmpty(t, samples)
}

func Test_Encode_FullFiles(t *testing.T) {
	_, _, enc, _ := extractorArgs(t, lang.JavaScript)

	for i, path := range testFilePaths {
		buf := assertReadFile(t, path)

		expected, err := enc.EncodeIdx(buf, path)
		require.NoError(t, err)

		toks, err := enc.Lexer.Lex(buf)
		require.NoError(t, err)

		for randIdx := 0; randIdx < len(toks); randIdx++ {
			actual, _ := encode(enc, path, toks, randIdx, len(expected))
			assert.Equal(t, expected, actual, "case %d, path %s", i, path)
		}
	}
}
