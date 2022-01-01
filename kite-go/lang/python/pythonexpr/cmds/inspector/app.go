package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

type session struct {
	LastUsed  time.Time
	Predicted predicted
}

type app struct {
	port      string
	m         sync.Mutex
	rm        pythonresource.Manager
	model     pythonexpr.Model
	templates *templateset.Set
	sessions  map[string]*session
}

func newApp(port string) *app {
	rmOpts := pythonresource.DefaultLocalOptions
	rmOpts.Dists = pythonresource.SmallOptions.Dists
	rm, errc := pythonresource.NewManager(rmOpts)
	fail(<-errc)

	opts := pythonmodels.DefaultOptions.ExprModelOpts
	model, err := pythonexpr.NewShardedModel(context.Background(), pythonmodels.DefaultOptions.ExprModelShards, opts)
	fail(err)

	fs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	templates := templateset.NewSet(fs, "templates", nil)

	return &app{
		port:      port,
		rm:        rm,
		model:     model,
		templates: templates,
		sessions:  make(map[string]*session),
	}
}

const initialBuffer = `
import requests

data = {content:"babar est content"}

requests.post($)
`

func (a *app) HandleHome(w http.ResponseWriter, r *http.Request) {
	err := a.templates.Render(w, "index.html", map[string]interface{}{
		"Buffer": initialBuffer,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *app) HandlePredict(w http.ResponseWriter, r *http.Request) {
	a.m.Lock()
	defer a.m.Unlock()

	type request struct {
		Buffer   string `json:"buffer"`
		Metaonly bool   `json:"metaonly"`
	}

	var rr request
	if err := json.NewDecoder(r.Body).Decode(&rr); err != nil {
		http.Error(w, fmt.Sprintf("error decoding json request: %v", err), http.StatusBadRequest)
		return
	}

	hash := pythoncode.CodeHash([]byte(rr.Buffer))

	var pred predicted
	if rr.Metaonly {
		var err error
		pred, err = a.newPrediction(hash, rr.Buffer, true)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to create meta info: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		s, ok := a.sessions[hash]
		if !ok {
			var err error
			s, err = a.newSession(hash, rr.Buffer)
			if err != nil {
				http.Error(w, fmt.Sprintf("error creating new session: %v", err), http.StatusBadRequest)
				return
			}
			a.sessions[hash] = s
		}
		s.LastUsed = time.Now()
		pred = s.Predicted
	}

	type response struct {
		Meta           string       `json:"meta"`
		PredictionTree string       `json:"prediction_tree"`
		Trace          string       `json:"trace"`
		Graph          *searchGraph `json:"graph"`
	}

	buf, err := json.Marshal(response{
		Meta:           pred.Meta,
		PredictionTree: pred.PredictionTree,
		Trace:          pred.Trace,
		Graph:          pred.Graph,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshaling response: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (a *app) HandleSearchNode(w http.ResponseWriter, r *http.Request) {
	a.m.Lock()
	defer a.m.Unlock()

	params := r.URL.Query()
	hash := params.Get("hash")
	nids := params.Get("node")

	nid, err := strconv.ParseInt(nids, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to parse node id '%s': %v", nids, err), http.StatusBadRequest)
		return
	}

	found, ok := a.sessions[hash]
	if !ok {
		http.Error(w, fmt.Sprintf("unable to find session for hash %s", hash), http.StatusNotFound)
		return
	}
	found.LastUsed = time.Now()

	node := found.Predicted.Graph.Nodes[nid]

	type link struct {
		HREF string
		Name string
	}

	var links []link
	for _, l := range node.SampleLinks {
		href := fmt.Sprintf("http://localhost%s/sample?hash=%s&name=%s&node=%v", a.port, l.Hash, l.Name, l.Node)
		links = append(links, link{
			HREF: href,
			Name: l.Name,
		})
	}

	err = a.templates.Render(w, "searchnode.html", map[string]interface{}{
		"Name":  nids,
		"Text":  node.Text,
		"Links": links,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *app) HandleSample(w http.ResponseWriter, r *http.Request) {
	a.m.Lock()
	defer a.m.Unlock()

	params := r.URL.Query()
	hash := params.Get("hash")
	name := params.Get("name")
	nids := params.Get("node")

	nid, err := strconv.ParseInt(nids, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to parse node id '%s': %v", nids, err), http.StatusBadRequest)
		return
	}

	found, ok := a.sessions[hash]
	if !ok {
		http.Error(w, fmt.Sprintf("unable to find session for hash %s", hash), http.StatusNotFound)
		return
	}
	found.LastUsed = time.Now()

	var sample renderedSample
	for _, s := range found.Predicted.Graph.Nodes[nid].Samples {
		if s.Name == name {
			sample = s
			break
		}
	}

	if sample.Name == "" {
		http.Error(w, fmt.Sprintf("unable to find sample with name %s and hash %s", name, hash), http.StatusNotFound)
		return
	}

	err = a.templates.Render(w, "sample.html", map[string]interface{}{
		"Name": sample.Name,
		"Body": sample.Graph.Body,
		"Head": sample.Graph.Head,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *app) cleanup() {
	for range time.NewTicker(time.Minute).C {
		func() {
			a.m.Lock()
			defer a.m.Unlock()
			for h, s := range a.sessions {
				if time.Since(s.LastUsed) > 10*time.Minute {
					delete(a.sessions, h)
				}
			}
		}()
	}
}

func (a *app) newSession(hash, buffer string) (*session, error) {
	pred, err := a.newPrediction(hash, buffer, false)
	if err != nil {
		return nil, err
	}
	return &session{
		LastUsed:  time.Now(),
		Predicted: pred,
	}, nil
}

func (a *app) newPrediction(hash, buffer string, metaonly bool) (predicted, error) {
	sc, err := newSrcCursor(buffer)
	if err != nil {
		return predicted{}, errors.Errorf("error making src cursor: %v", err)
	}

	pred, err := predict(a.rm, a.model, hash, sc, metaonly)
	if err != nil {
		return predicted{}, errors.Errorf("predict error: %v", err)
	}
	return pred, nil
}
