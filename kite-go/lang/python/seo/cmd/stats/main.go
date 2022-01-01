package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/seo"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	datadeps.Enable()

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	data, err := seo.Load(seo.DefaultDataPath)
	fail(err)

	docsMap := make(map[string][]string)
	for _, dist := range rm.Distributions() {
		syms, err := rm.CanonicalSymbols(dist)
		fail(err)
		for _, sym := range syms {
			if !data.CanonicalLinkPath(sym).Empty() {
				docs := rm.Documentation(sym)
				if docs == nil {
					log.Printf("no docs for %s", sym)
					continue
				}
				docsMap[docs.Text] = append(docsMap[docs.Text], sym.String())
			}
		}
	}

	f, err := os.Create("docs.json")
	fail(err)
	defer f.Close()
	fail(json.NewEncoder(f).Encode(docsMap))
}
