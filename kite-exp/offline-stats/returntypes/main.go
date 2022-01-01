package main

import (
	"fmt"
	"log"
	"strings"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

type distStats struct {
	name           string
	pubTotal       int
	pubHasReturn   int
	underTotal     int
	underHasReturn int
	dunderTotal    int
	nonFnHasReturn int
}

func (s distStats) Header() string {
	return fmt.Sprintf("name\tpublic\tunder\tdunder\tnon-function\n")
}

func (s distStats) String() string {
	return fmt.Sprintf("%s\t%d of %d\t%d of %d\t? of %d\t%d", s.name, s.pubHasReturn, s.pubTotal, s.underHasReturn, s.underTotal, s.dunderTotal, s.nonFnHasReturn)
}

func (s *distStats) Add(other distStats) {
	s.pubTotal += other.pubTotal
	s.pubHasReturn += other.pubHasReturn
	s.underTotal += other.underTotal
	s.underHasReturn += other.underHasReturn
	s.dunderTotal += other.dunderTotal
	s.nonFnHasReturn += other.nonFnHasReturn
}

func main() {
	var args struct {
		Manifest  string
		DistIndex string
	}
	arg.MustParse(&args)
	datadeps.Enable()

	opts, err := pythonresource.DefaultOptions.WithCustomPaths(args.Manifest, args.DistIndex)
	if err != nil {
		log.Fatalln(err)
	}

	opts.Manifest = opts.Manifest.Filter("SymbolGraph", "ReturnTypes") // names from pythonresource/internal/resources/resources.go
	rm, errc := pythonresource.NewManager(opts)
	if err := <-errc; err != nil {
		log.Fatalf("could not load resource manager: %s", err)
	}

	total := distStats{name: "*total*"}
	fmt.Println(total.Header())

	for _, dist := range opts.Manifest.Distributions() {
		s := distStats{name: dist.String()}
		syms, err := rm.CanonicalSymbols(dist)
		if err != nil {
			log.Printf("[ERROR] could not query canonical symbols for distribution %s", dist)
			continue
		}
		for _, sym := range syms {
			hasRet := len(rm.ReturnTypes(sym)) > 0

			path := sym.Path()
			switch {
			case rm.Kind(sym) != keytypes.FunctionKind:
				if hasRet {
					s.nonFnHasReturn++
				}
			case strings.HasPrefix(path.Parts[len(path.Parts)-1], "__"):
				s.dunderTotal++
			case strings.HasPrefix(path.Parts[len(path.Parts)-1], "_"):
				s.underTotal++
				if hasRet {
					s.underHasReturn++
				}
			default:
				s.pubTotal++
				if hasRet {
					s.pubHasReturn++
				}
			}
		}

		fmt.Println(s)
		total.Add(s)
	}

	fmt.Println(total)
}
