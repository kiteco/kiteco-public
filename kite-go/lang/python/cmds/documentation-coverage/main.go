package main

import (
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

type documap map[*pythonimports.Node]*pythondocs.LangEntity

func manualLoad(g *pythonimports.Graph, filePath string) documap {
	f, err := fileutil.NewCachedReader(filePath)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	decomp, err := gzip.NewReader(f)
	if err != nil {
		log.Fatalln(err)
	}
	dec := gob.NewDecoder(decomp)

	d := make(documap)
	for {
		var entity pythondocs.LangEntity
		err := dec.Decode(&entity)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln(err)
		}

		ident := entity.FullIdent()
		node, ok := g.FindByID(entity.NodeID)
		if !ok || node == nil {
			// fall back to find by CanonicalName
			var err error
			node, err = g.Find(ident)
			ok = err == nil
		}
		if !ok || node == nil {
			log.Printf("[loadEntityMap %s] identifier not found: %s", filePath, ident)
			continue
		}

		d[node] = &entity
	}
	return d
}

func autoLoad(g *pythonimports.Graph, htmlPath, stringsPath string) documap {
	opts := pythondocs.SearchOptions{
		DocPath:        htmlPath,
		DocstringsPath: stringsPath,
	}

	corpus, err := pythondocs.LoadCorpus(g, opts)
	if err != nil {
		log.Fatalln(err)
	}

	return corpus.Entities
}

func getDocstring(e *pythondocs.LangEntity) string {
	if e != nil && e.StructuredDoc != nil {
		return e.StructuredDoc.DescriptionHTML
	}
	return ""
}

func filterValid(d documap) documap {
	s := make(documap)
	for n, e := range d {
		if getDocstring(e) != "" {
			s[n] = e
		}
	}
	return s
}

// JSONOutput bundles documentation data for serialization
type JSONOutput struct {
	ID    int64
	Name  string
	Ident string
	HTML  string
}

func writeDocumap(d documap, fname string) {
	wJSONGZ, err := os.Create(fname)
	if err != nil {
		log.Fatalln(err)
	}
	defer wJSONGZ.Close()

	wJSON := gzip.NewWriter(wJSONGZ)
	defer wJSON.Close()

	enc := json.NewEncoder(wJSON)

	for n, e := range d {
		j := JSONOutput{
			ID:    n.ID,
			Name:  n.CanonicalName.String(),
			Ident: e.FullIdent(),
			HTML:  getDocstring(e),
		}
		enc.Encode(j)
	}
}

func main() {
	htmlPath := pythondocs.DefaultSearchOptions.DocPath
	oldStringsPath := "s3://kite-emr/datasets/documentation/python/2016-07-18_14-39-29-PM/pythondocstrings.gob.gz"
	newStringsPath := pythondocs.DefaultSearchOptions.DocstringsPath

	g, err := pythonimports.NewGraph(python.DefaultServiceOptions.ImportGraph)
	if err != nil {
		log.Fatalln(err)
	}

	html := filterValid(manualLoad(g, htmlPath))
	oldStrings := filterValid(manualLoad(g, oldStringsPath))
	newStrings := filterValid(manualLoad(g, newStringsPath))
	oldEffective := filterValid(autoLoad(g, htmlPath, oldStringsPath))
	newEffective := filterValid(autoLoad(g, htmlPath, newStringsPath))

	writeDocumap(html, "html.json.gz")
	writeDocumap(oldStrings, "old_strings.json.gz")
	writeDocumap(newStrings, "new_strings.json.gz")
	writeDocumap(oldEffective, "old_effective.json.gz")
	writeDocumap(newEffective, "new_effective.json.gz")

	log.Println("=======")
	log.Println("MISSING")
	log.Println("=======")
	for n, oldEntity := range oldEffective {
		if newEntity, ok := newEffective[n]; !ok { // node missing, check cases
			var foundIn []string
			if _, ok := html[n]; ok { // available through html; probably a merging issue
				foundIn = append(foundIn, "html")
			}
			if _, ok := oldStrings[n]; ok { // exists in old docstring data
				foundIn = append(foundIn, "old")
			}
			if _, ok := newStrings[n]; ok {
				foundIn = append(foundIn, "new")
			}
			log.Printf("%d %q %s", n.ID, n.CanonicalName.String(), strings.Join(foundIn, ";"))
		} else {
			if getDocstring(oldEntity) != getDocstring(newEntity) {
				log.Printf("%d %q mismatch", n.ID, n.CanonicalName.String())
			}
		}
	}

	log.Println("===")
	log.Println("NEW")
	log.Println("===")
	for n := range newEffective {
		if _, ok := oldEffective[n]; !ok {
			log.Printf("%d %q", n.ID, n.CanonicalName.String())
		}
	}
}
