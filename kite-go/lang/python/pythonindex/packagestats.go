package pythonindex

import (
	"encoding/json"
	"log"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/text"
)

func newPackageStatsIndex(graph pythonresource.Manager,
	path string, symbolToIdentCounts map[string][]*IdentCount) *index {

	f, err := awsutil.NewShardedFile(path)
	if err != nil {
		log.Fatalln("cannot open package stats file:", err)
	}

	invertedIndex := make(map[string][]*IdentCount)
	var m sync.Mutex

	err = awsutil.EMRIterateSharded(f, func(key string, value []byte) error {
		var stat pythoncode.PackageStats
		err := json.Unmarshal(value, &stat)
		if err != nil {
			return err
		}
		m.Lock()
		pkg := strings.ToLower(stat.Package)
		ic := &IdentCount{
			Ident:       stat.Package,
			Count:       stat.Count,
			ForcedCount: stat.Count,
		}
		invertedIndex[pkg] = append(invertedIndex[pkg], ic)

		// Index the package
		sym, err := graph.PathSymbol(pythonimports.NewDottedPath(stat.Package))
		if err == nil {
			path := sym.Canonical().PathString()
			symbolToIdentCounts[path] = append(symbolToIdentCounts[path], ic)

			// Index all members of the package
			if attrs, err := graph.Children(sym); err == nil {
				for _, attr := range attrs {
					if strings.HasPrefix(attr, "_") {
						continue
					}
					child, err := graph.ChildSymbol(sym, attr)
					if err != nil {
						continue
					}
					identifier := stat.Package + "." + attr
					indexNode(child.Canonical().PathString(), identifier, symbolToIdentCounts, invertedIndex)
				}
			}
		}

		for _, method := range stat.Methods {
			sym, err := graph.PathSymbol(pythonimports.NewDottedPath(method.Ident))
			if err != nil {
				continue
			}
			path := sym.Canonical().PathString()

			nc := &IdentCount{
				Ident:       method.Ident,
				Count:       method.Count,
				ForcedCount: method.Count,
			}
			// Make the identifier name in lowercase and split it by ".".
			parts := text.Uniquify(strings.Split(strings.ToLower(method.Ident), "."))
			for _, part := range parts {
				invertedIndex[part] = append(invertedIndex[part], nc)
			}
			symbolToIdentCounts[path] = append(symbolToIdentCounts[path], nc)

			if kind := graph.Kind(sym); kind != keytypes.ModuleKind && kind != keytypes.TypeKind {
				continue
			}
			if attrs, err := graph.Children(sym); err == nil {
				for _, attr := range attrs {
					if strings.HasPrefix(attr, "_") {
						continue
					}
					child, err := graph.ChildSymbol(sym, attr)
					if err != nil {
						continue
					}
					identifier := method.Ident + "." + attr
					indexNode(child.Canonical().PathString(), identifier, symbolToIdentCounts, invertedIndex)
				}
			}
		}

		m.Unlock()
		return nil
	})

	if err != nil {
		log.Fatalln("error in reading package stats:", err)
	}

	return &index{
		invertedIndex: invertedIndex,
	}
}
