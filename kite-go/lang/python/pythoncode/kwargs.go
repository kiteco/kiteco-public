package pythoncode

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

type processedKwargs struct {
	// Name is the name of the `**kwargs` parameter
	Name string
	// Kwargs are the possible `**kwargs`.
	Kwargs []processedKwarg
}

type processedKwarg struct {
	Name        string
	Probability float64
	Types       []kwargType
}

type pkwsByProb []processedKwarg

func (p pkwsByProb) Len() int           { return len(p) }
func (p pkwsByProb) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p pkwsByProb) Less(i, j int) bool { return p[i].Probability < p[j].Probability }

type kwargType struct {
	Type        string
	Probability float64
}

type kwtByProb []kwargType

func (p kwtByProb) Len() int           { return len(p) }
func (p kwtByProb) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p kwtByProb) Less(i, j int) bool { return p[i].Probability < p[j].Probability }

// KwargsOptions stores the options for retrieving possible **kwargs for a given function.
type KwargsOptions struct {
	CoverageKwargs float64
	MinUsageKwargs float64
	CoverageTypes  float64
	MinUsageTypes  float64
	MinDistance    int // Minimal Levenshtein distance to one of the kwargs already added (either from argSpec or free kwargs)
}

// DefaultKwargsOptions is the defalt options for retrieving possible **kwargs for a given function.
var DefaultKwargsOptions = KwargsOptions{
	CoverageKwargs: 0.90,
	MinUsageKwargs: 0.02,
	CoverageTypes:  0.9,
	MinUsageTypes:  0.1,
	MinDistance:    3,
}

// KwargsIndex stores information on the possible **kwargs that can be supplied to a given function.
type KwargsIndex struct {
	index map[*pythonimports.Node]processedKwargs
	opts  KwargsOptions
}

// NewKwargsIndex returns an empty index of possible **kwargs.
func NewKwargsIndex() *KwargsIndex {
	return &KwargsIndex{
		index: make(map[*pythonimports.Node]processedKwargs),
	}
}

// LoadKwargsIndex loads an index of possible **kwargs.
func LoadKwargsIndex(graph *pythonimports.Graph, opts KwargsOptions, data string) (*KwargsIndex, error) {
	f, err := fileutil.NewCachedReader(data)
	if err != nil {
		return nil, fmt.Errorf("error opening possbile kwargs data %s: %v", data, err)
	}
	defer f.Close()

	iter := awsutil.NewEMRIterator(f)

	index := &KwargsIndex{
		index: make(map[*pythonimports.Node]processedKwargs),
		opts:  opts,
	}
	for iter.Next() {
		var kws Kwargs
		if err := json.Unmarshal(iter.Value(), &kws); err != nil {
			return nil, fmt.Errorf("error unmarshalling possible kwargs for %s: %v", iter.Key(), err)
		}

		node, err := graph.Navigate(kws.AnyName)
		if node == nil || err != nil {
			continue
		}

		// transform counts to probabilities
		pkws := processedKwargs{
			Name: kws.Name,
		}

		var sumKwCounts int64
		for name, kw := range kws.Kwargs {
			pkw := processedKwarg{
				Name:        name,
				Probability: float64(kw.Count),
			}
			sumKwCounts += kw.Count

			var sumTypeCounts int64
			for typ, count := range kw.Types {
				if strings.HasPrefix(typ, "builtins.") {
					typ = strings.TrimPrefix(typ, "builtins.")
				}

				pkw.Types = append(pkw.Types, kwargType{
					Type:        typ,
					Probability: float64(count),
				})
				sumTypeCounts += count
			}

			invSumTypeCounts := 1. / float64(sumTypeCounts)
			for i := range pkw.Types {
				pkw.Types[i].Probability *= invSumTypeCounts
			}
			sort.Sort(sort.Reverse(kwtByProb(pkw.Types)))

			pkws.Kwargs = append(pkws.Kwargs, pkw)
		}

		invSumKwCounts := 1. / float64(sumKwCounts)
		for i := range pkws.Kwargs {
			pkws.Kwargs[i].Probability *= invSumKwCounts
		}
		sort.Sort(sort.Reverse(pkwsByProb(pkws.Kwargs)))

		index.index[node] = pkws
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("error reading possible kwargs data %s: %v", data, err)
	}

	return index, nil
}

// Index returns a map of all kwargs using thresholding provided by KwargsOptions
func (i KwargsIndex) Index() map[int64]*response.PythonKwargs {
	ret := make(map[int64]*response.PythonKwargs)
	for node := range i.index {
		kwargs := i.Kwargs(node)
		if kwargs != nil {
			ret[node.ID] = kwargs
		}
	}
	return ret
}

// Kwargs returns the possible **kwargs for a function.
// TODO(juan): move thresholding to load time
func (i KwargsIndex) Kwargs(node *pythonimports.Node) *response.PythonKwargs {
	pkws, found := i.index[node]
	if !found {
		return nil
	}

	var cdfKwargs float64
	var args []*response.PythonKwarg
	for _, pkw := range pkws.Kwargs {
		if pkw.Probability < i.opts.MinUsageKwargs {
			// can just break here since the possible kwargs are sorted
			// from highest probability to lowest.
			break
		}

		var cdfTypes float64
		var types []string
		for _, kwt := range pkw.Types {
			if kwt.Probability < i.opts.MinUsageTypes {
				// can just break here since the possible types are sorted
				// from highest probability to lowest.
				break
			}

			types = append(types, kwt.Type)
			if cdfTypes += kwt.Probability; cdfTypes > i.opts.CoverageTypes {
				break
			}
		}

		args = append(args, &response.PythonKwarg{
			Name:  pkw.Name,
			Types: types,
		})

		if cdfKwargs += pkw.Probability; cdfKwargs > i.opts.CoverageKwargs {
			break
		}
	}

	if len(args) < 1 {
		return nil
	}

	return &response.PythonKwargs{
		Name:   pkws.Name,
		Kwargs: args,
	}
}
