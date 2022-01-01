package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythondiffs"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

func main() {
	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	var funcDecorators []pythondiffs.FuncDecorator

	// lastKey captures the package name.
	var lastKey string
	for r.Next() {
		var fd pythondiffs.FuncDecorator
		err := json.Unmarshal(r.Value(), &fd)
		if err != nil {
			continue
		}
		if r.Key() != lastKey && len(funcDecorators) > 0 {
			sort.Sort(byDecorator(funcDecorators))
			// Emit data for a package
			err := emit(w, lastKey, funcDecorators)
			if err != nil {
				log.Println(err)
			}
			funcDecorators = funcDecorators[:0]
		}

		funcDecorators = append(funcDecorators, fd)
		lastKey = r.Key()
	}

	if len(funcDecorators) > 0 {
		// Emit data for a package
		sort.Sort(byDecorator(funcDecorators))
		err := emit(w, lastKey, funcDecorators)
		if err != nil {
			log.Println(err)
		}
	}

	if err := r.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}
}

func emit(w *awsutil.EMRWriter, key string, funcDecorators []pythondiffs.FuncDecorator) error {
	var lastFunc string
	var byFunc []pythondiffs.FuncDecorator

	// Emit DecoratorsProb by function.
	for i, fd := range funcDecorators {
		if lastFunc != fd.Func && len(byFunc) > 0 {
			model := decoratorsProbForPackageFunc(byFunc, key, lastFunc)
			out, err := json.Marshal(model)
			if err != nil {
				return err
			}
			w.Emit(key, out)
			byFunc = byFunc[:0]
		}
		byFunc = append(byFunc, funcDecorators[i])
		lastFunc = fd.Func
	}

	// Emit the last function.
	model := decoratorsProbForPackageFunc(byFunc, key, lastFunc)
	out, err := json.Marshal(model)
	if err != nil {
		return err
	}
	w.Emit(key, out)

	// Emit DecoratorsProb for the package.
	model = decoratorsProbForPackageFunc(funcDecorators, key, "")
	out, err = json.Marshal(model)
	if err != nil {
		return err
	}
	w.Emit(key, out)

	return nil
}

func decoratorsProbForPackageFunc(byFunc []pythondiffs.FuncDecorator, pkg, fun string) pythondiffs.DecoratorsProb {
	decoratorCount := make(map[string]*pythondiffs.DecoratorProb)

	for _, f := range byFunc {
		dc, found := decoratorCount[f.Decorator]
		if !found {
			dc = &pythondiffs.DecoratorProb{
				Canonical: f.Decorator,
			}
			decoratorCount[f.Decorator] = dc
		}
		dc.Prob++
	}

	model := pythondiffs.DecoratorsProb{
		Func:    fun,
		Package: pkg,
		Count:   len(byFunc),
	}

	for _, dc := range decoratorCount {
		dc.Prob = dc.Prob / float64(len(byFunc))
		model.DecorProbs = append(model.DecorProbs, dc)
	}

	return model
}

// byDecorator is a slice of FuncDecorator.
type byDecorator []pythondiffs.FuncDecorator

func (fds byDecorator) Len() int      { return len(fds) }
func (fds byDecorator) Swap(i, j int) { fds[j], fds[i] = fds[i], fds[j] }

// Sort the FuncDecorator obejcts according to 1) function name and 2) decorator name.
func (fds byDecorator) Less(i, j int) bool {
	if fds[i].Func == fds[j].Func {
		return fds[i].Decorator < fds[j].Decorator
	}
	return fds[i].Func < fds[j].Func
}
