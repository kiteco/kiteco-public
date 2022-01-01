package data

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpatterns"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

// Call to a particular function
type Call struct {
	Func       pythonpatterns.Symbol
	Positional []pythonpatterns.ExprSummary
	Keyword    map[string]pythonpatterns.ExprSummary
	Hash       string
}

// Calls is a group of calls
type Calls []Call

// Encode calls
func (c Calls) Encode() ([]byte, error) {
	return json.Marshal(c)
}

// Decode calls
func (c *Calls) Decode(buf []byte) error {
	return json.Unmarshal(buf, c)
}

// SampleTag implements pipeline.Sample
func (Calls) SampleTag() {}

func (c *Call) hash() pythonimports.Hash {
	var kws []string
	for k := range c.Keyword {
		kws = append(kws, k)
	}

	sort.Strings(kws)
	key := strings.Join([]string{
		fmt.Sprintf("%s-%d", c.Func.Dist.String(), c.Func.Path.Hash),
		fmt.Sprintf("%d", len(c.Positional)),
		fmt.Sprintf("%s", strings.Join(kws, ",")),
	}, ":")

	return pythonimports.Hash(spooky.Hash64([]byte(key)))
}

// SymPatterns bundles a symbol and a set of patterns
type SymPatterns struct {
	Sym      pythonresource.Symbol
	Patterns *pythonpatterns.Calls
}

// PatternsByHash map from symbol hash to patterns
type PatternsByHash map[pythonimports.Hash]SymPatterns

// LoadPatterns from the specified dir
func LoadPatterns(rm pythonresource.Manager, dir string) (PatternsByHash, error) {
	files, err := fileutil.ListDir(dir)
	if err != nil {
		return nil, err
	}

	ps := make(PatternsByHash)
	for _, f := range files {
		if strings.HasSuffix(f, "DONE") {
			continue
		}
		err := serialization.Decode(f, func(calls *pythonpatterns.Calls) error {
			sym, err := rm.NewSymbol(calls.Func.Dist, calls.Func.Path)
			if err != nil {
				log.Printf("error creating symbol from %s %s with kind %v: %v", calls.Func.Dist.String(), calls.Func.Path.String(), calls.Func.Kind, err)
				return nil
			}

			sym = sym.Canonical()

			if _, ok := ps[sym.Hash()]; ok {
				return fmt.Errorf("got multiple patterns for canonical sym %s from %v", sym, calls.Func)
			}

			ps[sym.Hash()] = SymPatterns{
				Sym:      sym,
				Patterns: calls,
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return ps, nil
}
