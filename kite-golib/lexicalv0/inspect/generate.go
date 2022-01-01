package inspect

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/text"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/utils"
)

const maxFileSize = 128000 // 128 kb

// CodeGenerator generates code for inspection
type CodeGenerator interface {
	// code, path, err
	Next() (string, string, error)
	Close()
}

// NewCodeGeneratorWithOpts ...
func NewCodeGeneratorWithOpts(language lexicalv0.LangGroup, local bool, cursor, localDataRoot string, seed int64) (CodeGenerator, error) {
	if local {
		return newLocalCodeGenerator(language, cursor, localDataRoot, seed)
	}
	return newTestSetGenerator(language, cursor, seed)
}

// NewCodeGenerator returns a CodeGenerator given a language and source of samples
func NewCodeGenerator(language lexicalv0.LangGroup, local bool, cursor string) (CodeGenerator, error) {
	return NewCodeGeneratorWithOpts(language, local, cursor, "", time.Now().UnixNano())
}

type testSetGenerator struct {
	language lexicalv0.LangGroup
	cursor   string
	data     []*dataset

	m    sync.Mutex
	rgen *rand.Rand
}

func (t *testSetGenerator) Next() (string, string, error) {
	t.m.Lock()
	defer t.m.Unlock()

	var code, path string
	for !admissibleCode(path, code, t.cursor, t.language) {
		d := t.data[t.rgen.Intn(len(t.data))]
		var full []byte
		var err error
		path, full, err = d.NewSample()
		if err != nil {
			return "", "", err
		}
		code, err = getFullCode(t.rgen, path, full, t.cursor)
		if err != nil {
			return "", "", err
		}
	}
	return code, path, nil
}

func (t *testSetGenerator) Close() {
	for _, d := range t.data {
		d.Close()
	}
}

func newTestSetGenerator(language lexicalv0.LangGroup, cursor string, seed int64) (*testSetGenerator, error) {
	rgen := rand.New(rand.NewSource(seed))

	data, err := readTestSets(rgen, language)
	if err != nil {
		return nil, errors.Wrapf(err, "language not supported for test set generation")
	}
	return &testSetGenerator{
		language: language,
		cursor:   cursor,
		data:     data,
		rgen:     rgen,
	}, nil
}

type dataset struct {
	files      []string
	current    int
	iterator   *awsutil.EMRIterator
	readcloser io.ReadCloser
	m          sync.Mutex
}

func readTestSets(rgen *rand.Rand, language lexicalv0.LangGroup) ([]*dataset, error) {
	sets := utils.DatasetForLang(utils.TestDataset, language)
	var datasets []*dataset
	for _, set := range sets {
		files, err := aggregator.ListDir(set)
		if err != nil {
			return nil, err
		}
		rgen.Shuffle(len(files), func(i, j int) {
			files[i], files[j] = files[j], files[i]
		})
		r, err := fileutil.NewCachedReader(files[0])
		if err != nil {
			return nil, err
		}
		d := dataset{
			files:      files,
			current:    0,
			iterator:   awsutil.NewEMRIterator(r),
			readcloser: r,
		}
		datasets = append(datasets, &d)
	}
	return datasets, nil
}

func (d *dataset) Close() {
	d.m.Lock()
	defer d.m.Unlock()
	d.readcloser.Close()
}

func (d *dataset) NewSample() (string, []byte, error) {
	d.m.Lock()
	defer d.m.Unlock()
	return d.newSampleLocked()
}

func (d *dataset) newSampleLocked() (string, []byte, error) {
	if d.iterator.Next() {
		return d.iterator.Key(), d.iterator.Value(), nil
	}
	if d.current == len(d.files)-1 {
		d.current = 0
	} else {
		d.current++
	}
	d.readcloser.Close()
	r, err := fileutil.NewCachedReader(d.files[d.current])
	if err != nil {
		return "", nil, err
	}
	d.iterator = awsutil.NewEMRIterator(r)
	d.readcloser = r
	return d.newSampleLocked()
}

type localCodeGenerator struct {
	language lexicalv0.LangGroup
	cursor   string
	paths    []string
	m        sync.Mutex
	rgen     *rand.Rand
}

func (l *localCodeGenerator) Next() (string, string, error) {
	l.m.Lock()
	defer l.m.Unlock()

	var code, path string
	for !admissibleCode(path, code, l.cursor, l.language) {
		path = l.paths[l.rgen.Intn(len(l.paths))]
		full, err := ioutil.ReadFile(path)
		if err != nil {
			return "", "", err
		}
		code, err = getFullCode(l.rgen, path, full, l.cursor)
		if err != nil {
			return "", "", err
		}
	}
	return code, path, nil
}

func (l *localCodeGenerator) Close() {
}

func newLocalCodeGenerator(language lexicalv0.LangGroup, cursor, localDataRoot string, seed int64) (*localCodeGenerator, error) {
	paths, err := utils.LocalFilesKiteco(language, localDataRoot)
	if err != nil {
		return nil, err
	}
	return &localCodeGenerator{
		language: language,
		cursor:   cursor,
		paths:    paths,
		rgen:     rand.New(rand.NewSource(seed)),
	}, nil
}

func getFullCode(rgen *rand.Rand, path string, code []byte, cursor string) (string, error) {
	tagLine := "random sample from " + path
	var tag string
	if ext := filepath.Ext(path); ext == ".html" {
		tag = fmt.Sprintf("<!-- %s -->\n\n", tagLine)
	} else {
		tag = fmt.Sprintf("%s %s\n\n", text.SingleLineCommentSymbols(lang.FromFilename(path))[0], tagLine)
	}
	tagged := tag + string(code)
	idx := len(tag) + rgen.Intn(len(code)+1)
	full := tagged[:idx] + cursor + tagged[idx:]
	return full, nil
}
