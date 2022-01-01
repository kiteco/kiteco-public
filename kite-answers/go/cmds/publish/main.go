package main

import (
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-answers/go/execution"
	"github.com/kiteco/kiteco/kite-answers/go/render"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/answers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/seo"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const postsPath = "/kite-answers/answers/posts"

var debug *log.Logger

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	datadeps.Enable()

	defer func() {
		if err := recover(); err != nil {
			log.Fatalln("fatal:", err)
		}
	}()

	if len(os.Args) < 2 {
		fatal(errors.New("usage: ./publish /path/to/posts/"))
	}

	log.SetPrefix("")
	log.SetFlags(0)

	ts := time.Now().UTC().Format("2006-01-02T15:04:05")

	debugFile, err := os.Create(ts + ".log")
	fatal(err)
	defer debugFile.Close()
	debug = log.New(debugFile, "", 0)

	// - collect all data
	log.Printf("collecting posts...")
	raws := make(map[string]render.Raw)
	fatal(filepath.Walk(os.Args[1], func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "could not read file %s", path)
		}

		raw, err := render.ParseRaw(buf)
		if err != nil {
			return errors.Wrapf(err, "could not parse front matter for file %s", path)
		}
		raws[path] = raw
		return nil
	}))

	// - slug validation
	log.Printf("generating title slugs...")
	allSlugs := make(map[string][]string)
	for path, raw := range raws {
		for _, slug := range raw.Slugs {
			allSlugs[slug] = append(allSlugs[slug], path)
		}
	}

	titleSlugs := make(map[string][]string)
	for path, raw := range raws {
		slug := strings.Join(strings.Split(strings.ToLower(strings.TrimSpace(raw.Title())), " "), "-")
		if slug == "" {
			debug.Printf("[slug-update] skipping %s no title)", path)
			continue
		}
		titleSlugs[slug] = append(titleSlugs[slug], path)
	}

	// - update slugs
	log.Printf("updating slugs...")
	for slug, paths := range titleSlugs {
		if len(paths) > 1 {
			for _, path := range paths {
				debug.Printf("[slug-update] skipping %s: (duplicate title slug %s)", path, slug)
			}
			continue
		}
		path := paths[0]

		if prevPaths := allSlugs[slug]; prevPaths != nil {
			if prevPaths[0] != paths[0] {
				debug.Printf("[slug-update] skipping %s (title slug %s duplicates existing slug)", path, slug)
			}
			continue
		}
		allSlugs[slug] = paths

		raw := raws[path]
		raw.Slugs = append([]string{slug}, raw.Slugs...)
		data, err := raw.Encode()
		fatal(err)
		fatal(ioutil.WriteFile(path, data, 0644))
		raws[path] = raw
	}

	// - indexing
	log.Printf("indexing...")
	sandbox := execution.NewManager(kitectx.Background())
	rm, errc := pythonresource.NewManager(pythonresource.DefaultLocalOptions.SymbolOnly())
	fatal(<-errc)

	seoData, err := seo.Load(seo.DefaultDataPath)
	fatal(err)

	links := make(map[pythonimports.Hash]map[string]string)
	idx := answers.Index{
		Slugs: make(map[string]int),
	}
	for path, raw := range raws {
		// - validate slugs
		var slugs []string
		for i, slug := range raw.Slugs {
			if len(allSlugs[slug]) > 1 {
				if i == 0 {
					debug.Printf("[validate-slugs] ignoring canonical slug %s: %s (duplicates another slug)", slug, path)
					break
				}
				debug.Printf("[validate-slugs] ignoring slug %s: %s (duplicates another slug)", slug, path)
				continue
			}
			slugs = append(slugs, slug)
		}
		if len(slugs) == 0 {
			debug.Printf("[validate-slugs] skipping %s (no canonical slug)", path)
			continue
		}

		log.Printf("rendering %s...", path)
		rendered, err := render.Render(kitectx.TODO(), sandbox, rm, raw)
		if err != nil {
			debug.Printf("[render] skipping %s (rendering error)\n%s", path, err)
			continue
		}

		for _, link := range rendered.Links {
			if link.Sym.Nil() {
				debug.Printf("[render] invalid link target %s in %s", link.Raw, path)
				continue
			}
			symPath := seoData.CanonicalLinkPath(link.Sym)
			if symPath.Empty() {
				debug.Printf("[seo] noindexed link target %s in %s", link.Raw, path)
				// still add it to the index, just using the pythonresource canonical path
				symPath = link.Sym.Canonical().Path()
			}

			pageLinks := links[symPath.Hash]
			if pageLinks == nil {
				pageLinks = make(map[string]string)
				links[symPath.Hash] = pageLinks
			}
			for _, slug := range slugs {
				pageLinks[slug] = raw.Title()
			}
		}

		pos := len(idx.Content)
		idx.Content = append(idx.Content, answers.Content{
			Rendered:  rendered,
			Canonical: slugs[0],
		})
		for _, slug := range slugs {
			idx.Slugs[slug] = pos
		}
	}

	idx.Links = make(map[pythonimports.Hash][]editorapi.AnswersLink)
	for hash, pageLinks := range links {
		pageLinksSlice := idx.Links[hash]
		for slug, title := range pageLinks {
			pageLinksSlice = append(pageLinksSlice, editorapi.AnswersLink{Slug: slug, Title: title})
		}
		// deterministic ordering
		sort.Slice(pageLinksSlice, func(i, j int) bool {
			return pageLinksSlice[i].Title < pageLinksSlice[j].Title
		})
		idx.Links[hash] = pageLinksSlice
	}

	log.Printf("writing index...")
	indexName := ts + ".json.gz"
	jsongzW, err := os.Create(indexName)
	fatal(err)
	defer jsongzW.Close()
	jsonW := gzip.NewWriter(jsongzW)
	defer jsonW.Close()
	fatal(json.NewEncoder(jsonW).Encode(idx))
	log.Printf("Done! Wrote index to %s.\nPlease update the S3 path in kite-go/lang/python/answers/index.go accordingly.", indexName)
}
