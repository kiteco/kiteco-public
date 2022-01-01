package main

import (
	"flag"
	"log"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/localcode"
)

func main() {
	var (
		uid      int64
		machine  string
		filename string
	)

	flag.Int64Var(&uid, "uid", 1, "")
	flag.StringVar(&machine, "machine", "machine", "")
	flag.StringVar(&filename, "filename", "test.txt", "")
	flag.Parse()

	python, err := pythonbatch.NewBuilderLoader(pythonbatch.DefaultOptions)
	if err != nil {
		log.Fatalln(err)
	}

	client := localcode.NewClient()
	localcode.RegisterLoader(lang.Python, python.Load)

	ctx, _ := client.CreateUserContext(uid, machine)
	latestTicker := time.NewTicker(time.Second)

	for {
		select {
		case <-latestTicker.C:
			_, err := ctx.ArtifactForFile(filename)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("got something :)")
			}
		}
	}
}
