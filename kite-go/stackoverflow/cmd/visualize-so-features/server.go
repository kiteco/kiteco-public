package main

import (
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/internal/search"
	"github.com/kiteco/kiteco/kite-golib/text"
	"github.com/kiteco/kiteco/kite-golib/tfidf"
)

var (
	tokenizer     = search.TextTokenizer
	processor     = search.TextProcessor
	codeTokenizer = text.CodeTokenizer{}
)

// Unwrapped version of SO Post so we can look at data in html
type soPost struct {
	ID               int64
	Title            string
	ShowTitle        bool
	Body             string
	Code             string
	ShowCode         bool
	Tags             string
	ShowTags         bool
	Score            int64
	ShowScore        bool
	ViewCount        int64
	ShowVC           bool
	AnswerCount      int64
	ShowAC           bool
	CommentCount     int64
	ShowCC           bool
	FavoriteCount    int64
	ShowFC           bool
	AcceptedAnswerID int64
	ShowAAID         bool
}

type soFeature struct {
	Name string
	Feat float64
}

type soPage struct {
	ID       int64
	URL      string
	Score    int
	Question []soPost
	Answers  []soPost
	Features []soFeature
}

type soSearchLog struct {
	Query   string
	Results []soPage
}

type soPageHandler struct {
	Logs       []soSearchLog
	NumQueries int
}

func (h soPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/logs.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, h)
}

func unwrapSOPost(post *stackoverflow.Post) soPost {

	code := text.CodeTokensFromHTML(post.GetBody())

	code = codeTokenizer.Tokenize(strings.Join(code, " "))
	return soPost{
		ID:               post.GetId(),
		Title:            strings.Join(processor.Apply(tokenizer.Tokenize(post.GetTitle())), " "),
		ShowTitle:        post.GetTitle() != "",
		Body:             strings.Join(processor.Apply(tokenizer.Tokenize(post.GetBody())), " "),
		Code:             strings.Join(code, " "),
		ShowCode:         len(code) > 0,
		Tags:             strings.Join(search.SplitTags(post.GetTags()), " "),
		ShowTags:         post.GetTags() != "",
		Score:            post.GetScore(),
		ShowScore:        post.GetScore() > 0,
		ViewCount:        post.GetViewCount(),
		ShowVC:           post.GetViewCount() > 0,
		AnswerCount:      post.GetAnswerCount(),
		ShowAC:           post.GetAnswerCount() > 0,
		CommentCount:     post.GetCommentCount(),
		ShowCC:           post.GetCommentCount() > 0,
		FavoriteCount:    post.GetFavoriteCount(),
		ShowFC:           post.GetFavoriteCount() > 0,
		AcceptedAnswerID: post.GetAcceptedAnswerId(),
		ShowAAID:         post.GetAcceptedAnswerId() > 0,
	}
}

func readSOPages(fileName string, featurers search.Featurers) []soSearchLog {
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var logs []soSearchLog
	decoder := json.NewDecoder(f)
	for {
		var l search.Log
		err = decoder.Decode(&l)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var soLog soSearchLog
		soLog.Query = strings.Join(processor.Apply(codeTokenizer.Tokenize(l.Query)), " ")
		for _, res := range l.Results {
			if res.Score < 6 {
				continue
			}
			var page soPage
			page.ID = res.ID
			page.URL = res.URL
			page.Score = res.Score
			page.Question = append(page.Question, unwrapSOPost(res.Page.GetQuestion().GetPost()))
			for _, ans := range res.Page.GetAnswers() {
				page.Answers = append(page.Answers, unwrapSOPost(ans.GetPost()))
			}
			feats := featurers.Features(l.Query, res)
			labels := featurers.Labels()
			page.Features = make([]soFeature, len(labels))
			for i, f := range feats {
				page.Features[i] = soFeature{
					Name: labels[i],
					Feat: f,
				}
			}
			soLog.Results = append(soLog.Results, page)
		}

		logs = append(logs, soLog)
	}
	return logs
}

func main() {
	var (
		start       = time.Now()
		logsPath    string
		dataDirPath string
	)
	flag.StringVar(&logsPath, "logs", "", "path containg []search.SearchLog")
	flag.StringVar(&dataDirPath, "data", "", "path to directory containing docCounts from running so-build-counters binary")
	flag.Parse()

	if logsPath == "" || dataDirPath == "" {
		flag.Usage()
		log.Fatal("logs and data all REQUIRED")
	}

	fDocCounts, err := os.Open(path.Join(dataDirPath, "docCounts"))
	if err != nil {
		log.Fatal(err)
	}
	defer fDocCounts.Close()
	decoder := gob.NewDecoder(fDocCounts)
	var docCounts map[string]*tfidf.IDFCounter
	err = decoder.Decode(&docCounts)
	if err != nil {
		log.Fatal(err)
	}

	featurers, err := search.NewFeaturers(docCounts)
	if err != nil {
		log.Fatal(err)
	}

	logs := readSOPages(logsPath, featurers)

	handler := soPageHandler{
		Logs:       logs,
		NumQueries: len(logs),
	}

	fmt.Println("Data loaded in ", time.Since(start))
	fmt.Println("serving on port: ", 8420)
	http.Handle("/", handler)
	log.Fatal(http.ListenAndServe(":8420", nil))
}
