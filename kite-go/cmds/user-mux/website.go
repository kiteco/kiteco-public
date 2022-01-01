package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type hybridWebsite struct {
	wp *httputil.ReverseProxy
	wd *httputil.ReverseProxy
}

func newHybridWebsite() *hybridWebsite {
	wpURL, err := url.Parse("https://XXXXXXX/")
	if err != nil {
		log.Fatalln(err)
	}

	wdURL, err := url.Parse("https://www.kite.com/")
	if err != nil {
		log.Fatalln(err)
	}

	return &hybridWebsite{
		wp: httputil.NewSingleHostReverseProxy(wpURL),
		wd: httputil.NewSingleHostReverseProxy(wdURL),
	}
}

func (h *hybridWebsite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case checkPathFor(r.URL.Path, "/docs/python/", "/python/docs/", "/invite", "/settings", "/static"):
		h.wd.ServeHTTP(w, r)
		return
	default:
		h.wp.ServeHTTP(w, r)
		return
	}
}

func checkPathFor(path string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
