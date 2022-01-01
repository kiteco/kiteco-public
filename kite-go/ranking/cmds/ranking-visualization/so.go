package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-golib/text"
)

type soPost struct {
	ID               int64
	ShowID           bool
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

func unwrapSOPost(post *stackoverflow.Post) soPost {
	tokenizer := text.NewHTMLTokenizer()
	code := strings.Join(text.CodeTokensFromHTML(post.GetBody()), " ")
	return soPost{
		ID:               post.GetId(),
		ShowID:           post.GetId() > 0,
		Title:            post.GetTitle(),
		ShowTitle:        post.GetTitle() != "",
		Body:             strings.Join(tokenizer.Tokenize(post.GetBody()), " "),
		Code:             code,
		ShowCode:         code != "",
		Tags:             post.GetTags(),
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

type soPage struct {
	Question soPost
	Answers  []soPost
}

func unwrapSOPage(page *stackoverflow.StackOverflowPage) *soPage {
	uPage := soPage{
		Question: unwrapSOPost(page.GetQuestion().GetPost()),
	}
	for _, ans := range page.GetAnswers() {
		uPage.Answers = append(uPage.Answers, unwrapSOPost(ans.GetPost()))
	}
	return &uPage
}

func loadEntriesSO(folder string, entries map[string]*entry) {
	in, err := os.Open(path.Join(folder, "test-results.json"))
	if err != nil {
		log.Fatal(err)
	}

	pages := make(map[int64]*soPage)

	decoder := json.NewDecoder(in)
	for {
		var result rankingResult
		err := decoder.Decode(&result)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if _, seen := seenQueries[result.QueryText]; !seen {
			queries = append(queries, &query{
				Text:  result.QueryText,
				Score: result.NDCG,
				URL:   "/viewer?query=" + result.QueryText,
			})
			seenQueries[result.QueryText] = struct{}{}
		}

		var snippets []snippet
		var page *soPage
		var exists bool

		for i, id := range result.SnapshotIDs {
			if page, exists = pages[id]; !exists {
				tempPages, err := soPagesClient.PostsByID([]int{int(id)})
				if err != nil {
					log.Println(err)
				}
				if len(tempPages) != 1 {
					continue
				}
				page = unwrapSOPage(tempPages[0])
				pages[id] = page
			}
			snippets = append(snippets, snippet{
				Title:         page.Question.Title,
				Rank:          i + 1,
				Label:         result.Labels[i],
				ExpectedRank:  result.ExpectedRank[i] + 1,
				Score:         result.Scores[i],
				Features:      result.Features[i],
				FeatureLabels: result.FeatureLabels,
				IsSO:          true,
				SOPage:        *page,
			})
		}
		entries[result.QueryText] = &entry{
			Score:    result.NDCG,
			Snippets: snippets,
		}
	}
}
