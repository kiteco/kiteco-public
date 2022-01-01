package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/codeexample"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

type snippet struct {
	SnippetID int64
	Title     string
	Prelude   string
	Code      string
}

func main() {
	var file string
	flag.StringVar(&file, "file", "", "code snippet emr file")
	flag.Parse()

	if file == "" {
		flag.Usage()
		return
	}

	in, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	r := awsutil.NewEMRReader(in)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	for {
		_, value, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var cs codeexample.CuratedSnippet
		err = json.Unmarshal(value, &cs)
		if err != nil {
			log.Fatal(err)
		}

		s := snippet{
			SnippetID: cs.Curated.Snippet.SnippetID,
			Title:     cs.Curated.Snippet.Title,
			Prelude:   cs.Curated.Snippet.Prelude,
			Code:      cs.Curated.Snippet.Code,
		}

		buf, err := json.Marshal(s)
		if err != nil {
			log.Fatal(err)
		}
		err = w.Emit(cs.Curated.Snippet.Package, buf)
		if err != nil {
			log.Fatal(err)
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}
