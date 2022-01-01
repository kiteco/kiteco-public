package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/alexflint/go-arg"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		In  string
		Out string
	}{}
	arg.MustParse(&args)

	r, err := fileutil.NewCachedReader(args.In)
	fail(err)
	defer r.Close()

	var mi pythonexpr.MetaInfo
	fail(json.NewDecoder(r).Decode(&mi))

	mi.Attr = mi.Attr.ForInference()
	mi.AttrBase = mi.AttrBase.ForInference()
	mi.Call = mi.Call.ForInference()
	mi.ProductionIndex = mi.ProductionIndex.ForInference()

	out, err := os.Create(args.Out)
	fail(err)
	defer out.Close()

	fail(json.NewEncoder(out).Encode(mi))
}
