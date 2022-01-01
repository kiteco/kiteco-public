package main

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"os"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

type byCount []*pythoncode.MethodStats

func (b byCount) Len() int           { return len(b) }
func (b byCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byCount) Less(i, j int) bool { return b[i].Count < b[j].Count }

func main() {
	r := csv.NewReader(os.Stdin)
	r.Comma = '\t'

	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	var lastKey string
	var exampleMap map[string]int

	for {
		values, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		key := values[0]
		value := values[1]

		if key != lastKey {
			// Key has changed, so lets summarize the counts for the last
			// key. Remember, the key is the module name.
			if len(exampleMap) > 0 {
				emitPackageStats(lastKey, exampleMap, w)
			}
			exampleMap = make(map[string]int)
		}

		// Count how many times each method occurs
		exampleMap[value]++
		lastKey = key
	}

	// emit last key
	if len(exampleMap) > 0 {
		emitPackageStats(lastKey, exampleMap, w)
	}
}

func emitPackageStats(key string, exampleMap map[string]int, w *awsutil.EMRWriter) {
	pkgSum := &pythoncode.PackageStats{
		Package: key,
	}

	for ident, count := range exampleMap {
		pkgSum.Count += count
		pkgSum.Methods = append(pkgSum.Methods, &pythoncode.MethodStats{
			Ident: ident,
			Count: count,
		})
	}

	sort.Sort(sort.Reverse(byCount(pkgSum.Methods)))
	buf, err := json.Marshal(pkgSum)
	if err != nil {
		log.Fatal(err)
	}

	err = w.Emit(pkgSum.Package, buf)
	if err != nil {
		log.Fatal(err)
	}
}
