//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."
package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/codegangsta/negroni"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

const (
	addr                      = ":8080"
	cursor                    = "$"
	removeNumLinesAfterCursor = 0
	allowContextAfterCursor   = false
	numExtsToShow             = 50
)

var (
	mutex        sync.Mutex
	cache        map[string]inspect.Sample
	activeSample inspect.Sample
	activeTime   int

	goPath        = os.Getenv("GOPATH")
	kitecoPath    = filepath.Join(goPath, "src/github.com/kiteco/kiteco")
	lexicalPath   = filepath.Join(kitecoPath, "kite-go/lang/lexical")
	inspectorPath = filepath.Join(lexicalPath, "cmds/inspector")

	templates     *templateset.Set
	codeGenerator inspect.CodeGenerator
	language      lexicalv0.LangGroup
	modelPaths    []string

	defaultConfig = predict.SearchConfig{
		Window:               64,
		TopK:                 10,
		TopP:                 1,
		MinP:                 0.02,
		BeamWidth:            5,
		Depth:                5,
		PrefixRegularization: 0.05,
	}
)

func main() {
	args := struct {
		Language string
		Local    bool
	}{
		Language: "",
		Local:    false,
	}
	arg.MustParse(&args)

	language = lexicalv0.MustLangGroupFromName(args.Language)

	modelPaths = readModelPaths()
	rand.Seed(time.Now().UTC().UnixNano())
	var err error
	codeGenerator, err = inspect.NewCodeGenerator(language, args.Local, cursor)
	if err != nil {
		log.Fatal(err)
	}
	defer codeGenerator.Close()
	cache = make(map[string]inspect.Sample)
	fs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	templates = templateset.NewSet(fs, "templates", nil)

	r := mux.NewRouter()

	r.HandleFunc("/", run)
	r.HandleFunc("/auto/", run)
	r.HandleFunc("/catalog/{hash}", load)
	r.HandleFunc("/save/{hash}", save)
	r.HandleFunc("/isFresh/", isFresh)
	r.HandleFunc("/beams/{hash}", detailedBeams)

	neg := negroni.New(
		midware.NewRecovery(),
		midware.NewLogger(contextutil.BasicLogger()),
		negroni.Wrap(r),
	)

	fmt.Printf("localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, neg))
}

func readModelPaths() []string {
	var modelPaths []string
	if language.Lexer == lang.JavaScript {
		modelPaths = append(modelPaths,
			"s3://kite-data/run-db/2020-07-20T03:38:09Z_lexical-model-experiments/out_javascript_prefix_suffix_context_1024_embedding_180_layer_4_head_6_vocab_20000_steps_50000_batch_70",
			"s3://kite-data/run-db/2020-02-05T20:25:06Z_lexical-model-experiments/out_javascript_lexical_context_512_embedding_180_layer_4_head_6_vocab_20000_steps_2000000_batch_5_slots_20",
		)
	}
	if lexicalv0.WebGroup.Equals(language) {
		modelPaths = append(modelPaths,
			"s3://kite-data/run-db/2020-09-29T03:48:57Z_lexical-model-experiments/out_text__javascript-jsx-vue-css-html-less-typescript-tsx_lexical_context_512_embedding_180_layer_4_head_6_vocab_13500_steps_25000_batch_160",
			"s3://kite-data/run-db/2020-10-07T18:29:44Z_lexical-model-experiments/out_text__javascript-jsx-vue-css-html-less-typescript-tsx_lexical_context_512_embedding_180_layer_4_head_6_vocab_13500_steps_25000_batch_160",
		)
	}
	if lexicalv0.JavaPlusPlusGroup.Equals(language) {
		modelPaths = append(modelPaths,
			"s3://kite-data/run-db/2020-10-08T05:09:31Z_lexical-model-experiments/out_text__java-scala-kotlin_lexical_context_512_embedding_180_layer_4_head_6_vocab_13500_steps_25000_batch_160",
		)
	}
	if lexicalv0.CStyleGroup.Equals(language) {
		modelPaths = append(modelPaths,
			"s3://kite-data/run-db/2020-10-09T05:43:16Z_lexical-model-experiments/out_text__c-cpp-objectivec-csharp_lexical_context_512_embedding_180_layer_4_head_6_vocab_20000_steps_25000_batch_160",
		)
	}
	if language.Lexer == lang.Text && language.Langs[0] == lang.Golang {
		modelPaths = append(modelPaths,
			"s3://kite-data/run-db/2020-09-15T21:49:44Z_lexical-model-experiments/out_text_lexical_context_1024_embedding_180_layer_4_head_6_vocab_13500_steps_25000_batch_80",
		)
	}
	if language.Lexer == lang.Golang {
		modelPaths = append(modelPaths,
			"s3://kite-data/run-db/2020-08-09T23:12:39Z_lexical-model-experiments/out_go_lexical_context_1024_embedding_180_layer_4_head_6_vocab_13500_steps_50000_batch_90",
		)
	}
	if language.Equals(lexicalv0.NewLangGroup(lang.Text, lang.Java)) {
		modelPaths = append(modelPaths,
			"s3://kite-data/run-db/2020-09-28T18:57:46Z_lexical-model-experiments/out_text__java_lexical_context_512_embedding_180_layer_4_head_6_vocab_13500_steps_25000_batch_160",
		)
	}

	if language.Lexer != lang.Text {
		path := fileutil.Join("s3://kite-local-pipelines/lexical-inspector/models", language.Name())
		reader, err := fileutil.NewReader(path)
		if err != nil {
			log.Fatal(err)
		}
		defer reader.Close()
		contents, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Fatal(err)
		}
		lines := strings.Split(string(contents), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			modelPaths = append(modelPaths, line)
		}
	}
	return modelPaths
}

func run(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	auto := strings.Contains(r.URL.Path, "auto")
	query := newQuery(r, auto)
	sample, err := inspect.Inspect(query)
	if err != nil {
		show(w, displayError(query, err, auto))
		return
	}
	showSample(w, sample, auto)
	updateState(sample, auto)
}

func detailedBeams(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	key := getKey(r)

	sample, ok := cache[key]
	if !ok {
		http.Error(w, fmt.Sprintf("sample for key %s not in cache", key), http.StatusBadRequest)
		return
	}

	showLayers := r.FormValue("showlayers") != ""

	showDetailedBeams(w, sample, showLayers, false)
}

func load(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	key := getKey(r)
	sample, err := inspect.Load(key)
	if err != nil {
		show(w, displayError(sample.Query, err, false))
		return
	}
	if !sample.Query.Language.Equals(language) {
		err := errors.New("sample language does not match session language")
		show(w, displayError(sample.Query, err, false))
		return
	}
	updateModelPaths(sample.Query.ModelPath)
	showSample(w, sample, false)
	updateState(sample, false)
}

func save(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	sample, ok := cache[getKey(r)]
	if !ok {
		log.Fatal(errors.New("key not found in cache"))
	}
	key, err := inspect.Save(sample)
	if err != nil {
		log.Fatal(err)
	}
	if key != getKey(r) {
		log.Fatal(errors.New("key mismatch"))
	}
	link := fmt.Sprintf("localhost%s/catalog/%s", addr, key)
	err = templates.Render(w, "save.html", map[string]string{"Key": key, "Link": link})
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to render index template: %v", err), http.StatusInternalServerError)
	}
}

func newQuery(r *http.Request, auto bool) inspect.Query {
	if r.FormValue("window") == "" {
		return firstQuery(auto)
	}
	config := predict.SearchConfig{
		Window:               readInt(r.FormValue("window")),
		TopK:                 readInt(r.FormValue("topk")),
		TopP:                 float32(readInt(r.FormValue("topp"))) / 100,
		MinP:                 float32(readFloat(r.FormValue("minp"))),
		BeamWidth:            readInt(r.FormValue("beamwidth")),
		Depth:                readInt(r.FormValue("depth")),
		PrefixRegularization: float32(readFloat(r.FormValue("prefixreg"))),
	}
	code, path := getCode(r, auto)
	return inspect.Query{
		Path:                      path,
		Cursor:                    cursor,
		ModelPath:                 r.FormValue("modelpath"),
		Code:                      code,
		Config:                    config,
		Language:                  language,
		RemoveNumLinesAfterCursor: removeNumLinesAfterCursor,
		AllowContextAfterCursor:   allowContextAfterCursor,
	}
}

func firstQuery(auto bool) inspect.Query {
	code := activeSample.Query.Code
	path := displayPath(activeSample.Query.Path)
	if !auto {
		var err error
		code, path, err = codeGenerator.Next()
		if err != nil {
			log.Fatal(err)
		}
	}
	config := predict.SearchConfig{
		Window:               defaultConfig.Window,
		TopK:                 defaultConfig.TopK,
		TopP:                 defaultConfig.TopP,
		MinP:                 defaultConfig.MinP,
		BeamWidth:            defaultConfig.BeamWidth,
		Depth:                defaultConfig.Depth,
		PrefixRegularization: defaultConfig.PrefixRegularization,
	}
	return inspect.Query{
		Path:                      path,
		Cursor:                    cursor,
		ModelPath:                 modelPaths[0],
		Code:                      truncate(code, cursor),
		Config:                    config,
		Language:                  language,
		RemoveNumLinesAfterCursor: removeNumLinesAfterCursor,
		AllowContextAfterCursor:   allowContextAfterCursor,
	}
}

func getCode(r *http.Request, auto bool) (string, string) {
	if auto {
		return activeSample.Query.Code, displayPath(activeSample.Query.Path)
	}
	if r.FormValue("mode") == "Predict" {
		// TODO (juan): use the real path
		return r.FormValue("code"), r.FormValue("filename")
	}
	code, path, err := codeGenerator.Next()
	if err != nil {
		log.Fatal(err)
	}
	return truncate(code, cursor), path
}

func getKey(r *http.Request) string {
	return strings.Split(r.URL.Path, "/")[2]
}

func readInt(s string) int {
	value, err := strconv.Atoi(s)
	if err != nil {
		log.Fatal(err)
	}
	return value
}

func readFloat(s string) float64 {
	value, err := strconv.ParseFloat(s, 32)
	if err != nil {
		log.Fatal(err)
	}
	return value
}

func show(w http.ResponseWriter, d display) {
	err := templates.Render(w, "index.html", d)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to render index template: %v", err), http.StatusInternalServerError)
	}
}

func showSample(w http.ResponseWriter, sample inspect.Sample, auto bool) {
	display := displaySample(sample, true, auto)
	show(w, display)
}

func showDetailedBeams(w http.ResponseWriter, sample inspect.Sample, showLayers, auto bool) {
	d := displaySample(sample, showLayers, auto)

	err := templates.Render(w, "beams.html", d)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to render beams template: %v", err), http.StatusInternalServerError)
	}
}

func updateState(sample inspect.Sample, auto bool) {
	if auto {
		return
	}
	key, err := inspect.Key(sample)
	if err != nil {
		log.Fatal(err)
	}
	cache[key] = sample
	activeSample = sample
	activeTime = int(time.Now().Unix())
}

func updateModelPaths(path string) {
	for _, modelPath := range modelPaths {
		if modelPath == path {
			return
		}
	}
	modelPaths = append(modelPaths, path)
}

func isFresh(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	w.Write([]byte(strconv.Itoa(activeTime)))
}

func readCheckbox(s string) bool {
	return s == "on"
}
