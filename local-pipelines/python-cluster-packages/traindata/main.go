package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

type coOccurrence struct {
	Pkg1 string
	Pkg2 string
}

type scoredCoOccurrence struct {
	Pkg1  string `json:"pkg1"`
	Pkg2  string `json:"pkg2"`
	Score int    `json:"score"`
}

type scoredName struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getPackageScores(endpoint string) []scoredName {
	url, err := url.Parse(endpoint)
	fail(err)
	url, err = url.Parse("symbol/packages")
	fail(err)

	resp, err := http.Get(url.String())
	fail(err)
	defer resp.Body.Close()

	var res []scoredName
	fail(json.NewDecoder(resp.Body).Decode(&res))
	return res
}

func getImports(endpoint string, symbol string, context pythoncode.SymbolContext, limit int) [][]string {
	request := struct {
		Symbol  string `json:"symbol"`
		Context string `json:"context"`
		Limit   int    `json:"limit"`
	}{
		Symbol:  symbol,
		Context: string(context),
		Limit:   limit,
	}

	var buf bytes.Buffer
	fail(json.NewEncoder(&buf).Encode(request))

	url, err := url.Parse(endpoint)
	fail(err)
	url, err = url.Parse("symbol/imports")
	fail(err)

	resp, err := http.Post(url.String(), "application/json", &buf)
	fail(err)
	defer resp.Body.Close()

	type response struct {
		Imports [][]string `json:"imports"`
	}

	var res response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return make([][]string, 0, 0)
	}

	// trim everything after the dot
	var trimmed [][]string
	for _, imps := range res.Imports {
		var pkgs []string
		for _, imp := range imps {
			pkgs = append(pkgs, pythonimports.NewDottedPath(imp).Head())
		}
		trimmed = append(trimmed, pkgs)
	}

	return trimmed
}

func main() {
	args := struct {
		Endpoint    string
		MaxPackages int
		MaxGo       int
		Out         string
		MaxFiles    int
	}{
		Endpoint:    "http://ml-training-0.kite.com:3039",
		MaxPackages: 200,
		MaxGo:       2,
		Out:         "cooccurs.json",
		MaxFiles:    500,
	}

	arg.MustParse(&args)

	pkgs := getPackageScores(args.Endpoint)

	if args.MaxPackages > 0 && len(pkgs) > args.MaxPackages {
		pkgs = pkgs[:args.MaxPackages]
	}

	keep := make(map[string]bool)
	for _, pkg := range pkgs {
		keep[pkg.Name] = true
	}

	jobs := make([]workerpool.Job, 0, len(pkgs))
	coOccurs := make(chan coOccurrence)
	for _, p := range pkgs {
		pkg := p.Name
		job := func() error {
			for _, imps := range getImports(args.Endpoint, pkg, pythoncode.SymbolContextImport, args.MaxFiles) {
				for _, p1 := range imps {
					if !keep[p1] {
						continue
					}

					for _, p2 := range imps {
						if p1 == p2 {
							continue
						}

						if !keep[p2] {
							continue
						}

						coOccurs <- coOccurrence{
							Pkg1: p1,
							Pkg2: p2,
						}
						coOccurs <- coOccurrence{
							Pkg1: p2,
							Pkg2: p1,
						}
					}
				}
			}
			return nil
		}
		jobs = append(jobs, job)
	}

	pool := workerpool.New(args.MaxGo)
	pool.Add(jobs)
	go func() {
		pool.Wait()
		pool.Stop()
		close(coOccurs)
	}()

	scores := make(map[coOccurrence]int)
	for c := range coOccurs {
		scores[c]++
	}

	var results []scoredCoOccurrence
	for c, s := range scores {
		results = append(results, scoredCoOccurrence{
			Pkg1:  c.Pkg1,
			Pkg2:  c.Pkg2,
			Score: s,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Pkg1 == results[j].Pkg1 {
			return results[i].Pkg2 < results[j].Pkg2
		}
		return results[i].Pkg1 < results[j].Pkg2
	})

	out, err := os.Create(args.Out)
	fail(err)
	defer out.Close()

	fail(json.NewEncoder(out).Encode(results))
}
