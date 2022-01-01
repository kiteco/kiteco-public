package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

const idleSession = 10 * time.Minute

type session struct {
	Task      string
	Hash      string
	Completed *completedTask
	LastUsed  time.Time
}

type app struct {
	m         sync.Mutex
	templates *templateset.Set
	rm        pythonresource.Manager
	sessions  []*session
	mi        pythonexpr.MetaInfo
}

func newApp() *app {
	rmOpts := pythonresource.DefaultLocalOptions
	rmOpts.Dists = pythonresource.SmallOptions.Dists
	rm, errc := pythonresource.NewManager(rmOpts)
	fail(<-errc)

	app := new(app)
	app.rm = rm

	app.templates = templateset.NewSet(
		&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}, "templates", nil)

	mi, err := pythonexpr.NewMetaInfo(pythonmodels.DefaultOptions.ExprModelShards[0].ModelPath)
	fail(err)
	app.mi = mi

	return app
}

const initialBuffer = `
import numpy as np
start = -1
end = 1
n = 10
x = np.linspace(start, end, num=n)
`

func (a *app) HandleHome(w http.ResponseWriter, r *http.Request) {
	err := a.templates.Render(w, "index.html", map[string]interface{}{
		"Buffer": initialBuffer,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *app) HandleBuildSamples(w http.ResponseWriter, r *http.Request) {
	a.m.Lock()
	defer a.m.Unlock()

	type request struct {
		Task string `json:"task"`
		Src  string `json:"src"`
	}

	var sr request
	if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
		log.Println("err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h := hash(sr.Src)

	var found *session
	var sessions []*session
	for _, s := range a.sessions {
		if s.Hash == h && s.Task == sr.Task {
			s.LastUsed = time.Now()
			found = s
		}
		if time.Since(s.LastUsed) < idleSession {
			sessions = append(sessions, s)
		}
	}

	a.sessions = sessions

	if found == nil {
		sc, err := newSrcCursor(sr.Src)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ct, err := worker{
			RM: a.rm,
			MI: a.mi,
		}.doTask(sc, sr.Task)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		found = &session{
			Hash:      h,
			Task:      sr.Task,
			Completed: ct,
			LastUsed:  time.Now(),
		}
		a.sessions = append(a.sessions, found)
	}

	type link struct {
		Task string `json:"task"`
		Name string `json:"name"`
		Hash string `json:"hash"`
	}

	type resp struct {
		MetaInfo string `json:"metainfo"`
		Links    []link `json:"links"`
	}

	var links []link
	for _, s := range found.Completed.Samples {
		links = append(links, link{
			Task: sr.Task,
			Name: s.Name,
			Hash: h,
		})
	}

	buf, err := json.Marshal(resp{
		MetaInfo: found.Completed.MetaInfo,
		Links:    links,
	})
	if err != nil {
		log.Println(err)
		http.Error(w, fmt.Sprintf("error marshalling links: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (a *app) HandleSample(w http.ResponseWriter, r *http.Request) {
	a.m.Lock()
	defer a.m.Unlock()

	params := r.URL.Query()
	task := params.Get("task")
	hash := params.Get("hash")
	name := params.Get("name")

	var found *session
	for _, s := range a.sessions {
		if s.Hash == hash && s.Task == task {
			found = s
			break
		}
	}

	if found == nil {
		http.Error(w, fmt.Sprintf("unable to find session for task %s and hash %s", task, hash), http.StatusNotFound)
		return
	}
	found.LastUsed = time.Now()

	var sample *renderedSample
	for _, s := range found.Completed.Samples {
		if s.Name == name {
			sample = s
			break
		}
	}

	if sample == nil {
		http.Error(w, fmt.Sprintf("unable to find sample with name %s for task %s and hash %s", name, task, hash), http.StatusNotFound)
		return
	}

	err := a.templates.Render(w, "sample.html", map[string]interface{}{
		"Name": sample.Name,
		"Task": task,
		"Body": sample.Graph.Body,
		"Head": sample.Graph.Head,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func hash(s string) string {
	return pythoncode.CodeHash([]byte(s))
}
