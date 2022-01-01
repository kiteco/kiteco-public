package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/serialization"
	"github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/type-induction/data"
)

const (
	funcSmoothing     = .0001
	attrSmoothing     = .0001
	variableSmoothing = .0001
	eps               = 1e-6
)

var (
	logger io.Writer = os.Stdout
	level            = logLevelWarn
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func readSamples(fileName string, maxSamples int) []data.Sample {
	var all []data.Sample
	err := serialization.Decode(fileName, func(sample *data.Sample) {
		if len(all) < maxSamples {
			all = append(all, *sample)
		} else {
			return
		}
	})
	fail(err)
	return all
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func saveModelByPkg(model *model, dir string) {
	results := make(map[string][]pythoncode.EMReturnTypes)
	for _, f := range model.funcs {
		pkg := f.Sym.Path().Head()

		rets := make([]*Type, 0, len(f.Return))
		for t := range f.Return {
			rets = append(rets, t)
		}

		sort.Slice(rets, func(i, j int) bool {
			ti, tj := rets[i], rets[j]
			return f.Return[ti] > f.Return[tj]
		})

		filtered := make([]pythoncode.EMType, 0, len(rets))
		for _, t := range rets {
			filtered = append(filtered, pythoncode.EMType{
				Sym:  t.Sym.Path(),
				Dist: t.Dist,
				Prob: f.Return[t],
			})
		}
		results[pkg] = append(results[pkg], pythoncode.EMReturnTypes{
			Func:        f.Sym.Path(),
			Dist:        f.Dist,
			ReturnTypes: filtered,
		})
	}

	for p, r := range results {
		outFileName := pathForPkg(dir, p)
		out, err := fileutil.NewBufferedWriter(outFileName)
		fail(err)
		fail(json.NewEncoder(out).Encode(r))
		out.Close()
	}
}

func saveAttrDistByPkg(model *model, attrsDir string, pkgFilter map[string]bool) {
	pkgToTypeAndAttrs := make(map[string][]TypeAndAttrs)
	for _, t := range model.types {
		p := t.Pkg
		if pkgToTypeAndAttrs[p] == nil {
			pkgToTypeAndAttrs[p] = make([]TypeAndAttrs, 0)
		}
		pkgToTypeAndAttrs[p] = append(pkgToTypeAndAttrs[p], TypeAndAttrs{
			Type:  data.NewSymbol(t.Sym),
			Attrs: t.Attrs,
		})
	}

	for p, typeAndAttrs := range pkgToTypeAndAttrs {
		fileName := pathForPkg(attrsDir, p)
		if pkgFilter[p] && !fileExists(fileName) {
			distf, err := fileutil.NewBufferedWriter(fileName)
			fail(err)
			fail(json.NewEncoder(distf).Encode(typeAndAttrs))
			distf.Close()
		}
	}
}

func pathForPkg(dir string, pkg string) string {
	return fileutil.Join(dir, fmt.Sprintf("%s.json", pkg))
}

func main() {
	args := struct {
		Packages     string
		SamplesDir   string
		AttrDist     string
		ModelPath    string
		MaxSamples   int
		MinSamples   int
		MinCount     int
		Steps        int
		LossInterval int
		Together     bool
		Dependency   string
		MaxAttrs     int
	}{
		// we need to be careful setting min count because
		// if the type we are trying to learn only gets returned from
		// functions then we will not even consider it as a return type
		MinCount:     5,
		Dependency:   "deps.json",
		MaxSamples:   100000,
		MinSamples:   10,
		MaxAttrs:     20,
		Steps:        1000,
		LossInterval: 10,
		Together:     false,
	}
	arg.MustParse(&args)

	pkgs, err := traindata.LoadPackageList(args.Packages)
	fail(err)

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	r, err := fileutil.NewCachedReader(args.Dependency)
	fail(err)
	defer r.Close()
	var deps map[string][]string
	fail(json.NewDecoder(r).Decode(&deps))

	if args.Together {
		start := time.Now()

		allPackages := make(map[string]bool)
		var allData []data.Sample
		for _, pkg := range pkgs {
			allPackages[pkg] = true
			sampleFileName := pathForPkg(args.SamplesDir, pkg)
			if !fileExists(sampleFileName) {
				fmt.Printf("Skipping adding data for package %s, train data not found\n", pkg)
				continue
			}
			allData = append(allData, readSamples(sampleFileName, args.MaxSamples)...)
		}

		allPackages["__builtin__"] = true
		model := newModel(allData, rm, allPackages, args.MinCount, loadTypeAndAttrs(rm, allPackages, args.AttrDist), args.MaxAttrs)
		fmt.Printf("Done preparing model, took %v\n", time.Since(start))

		start = time.Now()
		fmt.Println("starting training at", start)
		model.EM(args.Steps, args.LossInterval)

		saveModelByPkg(model, args.ModelPath)
		saveAttrDistByPkg(model, args.AttrDist, allPackages)

		fmt.Printf("Done. Took %v, trained packages toegther from %v\n", time.Since(start), args.Packages)
		return
	}

	for _, pkg := range pkgs {
		// Getting training samples
		start := time.Now()

		sampleFileName := pathForPkg(args.SamplesDir, pkg)
		if !fileExists(sampleFileName) {
			fmt.Printf("Skipping training for package %s, train data not found\n", pkg)
			continue
		}

		outFileName := pathForPkg(args.ModelPath, pkg)
		if fileExists(outFileName) {
			fmt.Printf("Skipping training for package %s, model already exists\n", pkg)
			continue
		}

		samples := readSamples(sampleFileName, args.MaxSamples)
		fmt.Printf("Took %v to retrive training samples for pkg %s\n", time.Since(start), pkg)
		start = time.Now()
		allPackages := make(map[string]bool)
		for _, d := range deps[pkg] {
			allPackages[d] = true
		}
		allPackages[pkg] = true
		allPackages["__builtin__"] = true

		pretrainedAttrs := loadTypeAndAttrs(rm, allPackages, args.AttrDist)
		model := newModel(samples, rm, allPackages, args.MinCount, pretrainedAttrs, args.MaxAttrs)
		fmt.Printf("done preparing model, took %v\n", time.Since(start))

		start = time.Now()
		fmt.Println("starting training at", start)
		model.EM(args.Steps, args.LossInterval)

		saveModelByPkg(model, args.ModelPath)
		saveAttrDistByPkg(model, args.AttrDist, map[string]bool{pkg: true})

		fmt.Printf("Done training for %v, took %v, saved model in %s/%s.json\n", pkg, time.Since(start), args.ModelPath, pkg)
	}
}
