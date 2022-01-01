package main

import (
	"io/ioutil"
	"log"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
)

func main() {
	var extensions []string
	for _, language := range lexicalv0.AllLangsGroup.Langs {
		for _, ext := range language.Extensions() {
			extensions = append(extensions, "."+ext)
		}
	}
	extensions = append(extensions, ".md")

	data := []byte(strings.Join(extensions, "\n"))
	err := ioutil.WriteFile("extensions", data, 0600)
	if err != nil {
		log.Fatal(err)
	}
}
