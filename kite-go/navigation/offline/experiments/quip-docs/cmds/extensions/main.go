package main

import (
	"io/ioutil"
	"log"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
)

func main() {
	args := struct {
		Extensions string
	}{}
	arg.MustParse(&args)

	var extensions []string
	for _, language := range lexicalv0.AllLangsGroup.Langs {
		for _, ext := range language.Extensions() {
			extensions = append(extensions, "."+ext)
		}
	}
	extensions = append(extensions, ".md")

	data := []byte(strings.Join(extensions, "\n"))
	err := ioutil.WriteFile(args.Extensions, data, 0600)
	if err != nil {
		log.Fatal(err)
	}
}
