package main

import (
	"flag"
	"log"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/localcode"
)

func main() {
	var (
		hostPort string
		uid      int64
		machine  string
		filename string
	)

	flag.StringVar(&hostPort, "hostPort", "", "")
	flag.Int64Var(&uid, "uid", 0, "")
	flag.StringVar(&machine, "machine", "", "")
	flag.StringVar(&filename, "filename", "", "")
	flag.Parse()

	python, err := pythonbatch.NewBuilderLoader(pythonbatch.DefaultOptions)
	if err != nil {
		log.Fatalln(err)
	}

	localcode.RegisterLoader(lang.Python, python.Load)

	client, err := localcode.NewStandaloneClient(hostPort)
	if err != nil {
		log.Fatalln(err)
	}

	obj, err := client.FindArtifact(uid, machine, filename)
	if err != nil {
		log.Fatalln(err)
	}

	index, ok := obj.(*pythonlocal.SymbolIndex)
	if !ok {
		log.Fatalln("got unexpected artifact")
	}

	log.Printf("%+v", index)

	// Do things here....
}
