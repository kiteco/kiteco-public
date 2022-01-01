package main

import (
	"fmt"
	"net/url"
	"path"
)

const (
	xmlns    = "http://www.sitemaps.org/schemas/sitemap/0.9"
	urlLimit = 49000 // according to the spec: 50000
)

var (
	emptySMI = sitemapIndex{Xmlns: xmlns}
	emptySM  = sitemap{Xmlns: xmlns}
)

type urlSet = map[string]struct{}

type sitemapIndex struct {
	XMLName     struct{}           `xml:"sitemapindex"`
	Xmlns       string             `xml:"xmlns,attr"`
	SitemapURLs []urlEntry         `xml:"sitemap"`
	Sitemaps    map[string]sitemap `xml:"-"`
}

type sitemap struct {
	XMLName struct{}   `xml:"urlset"`
	Xmlns   string     `xml:"xmlns,attr"`
	URLs    []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc        string `xml:"loc"`
	ChangeFreq string `xml:"changefreq,omitempty"`
}

func newSitemapIndex() sitemapIndex {
	smi := emptySMI
	smi.Sitemaps = make(map[string]sitemap)
	return smi
}

func (smi sitemapIndex) Append(sitemapRoot url.URL, fnameTpl string, urls urlSet) sitemapIndex {
	sm := emptySM
	emit := func() {
		// set filename
		fname := fmt.Sprintf(fnameTpl, len(smi.Sitemaps))
		smi.Sitemaps[fname] = sm

		// compute remote URL
		smURL := sitemapRoot
		smURL.Path = path.Join(smURL.Path, fname)
		smi.SitemapURLs = append(smi.SitemapURLs, urlEntry{Loc: smURL.String()})

		// reset state
		sm = emptySM
	}

	for url := range urls {
		if len(sm.URLs) == urlLimit {
			emit()
		}
		sm.URLs = append(sm.URLs, urlEntry{
			Loc: url,
			// TODO(naman) do we need/want this?
			ChangeFreq: "monthly",
		})
	}
	emit()

	return smi
}
