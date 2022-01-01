package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/kiteco/kiteco/kite-golib/diskmap"
)

func main() {
	var path string
	var key string

	flag.StringVar(&path, "path", "", "path to diskmap")
	flag.StringVar(&key, "key", "", "key to lookup")
	flag.Parse()

	dm, err := diskmap.NewMap(path)
	if err != nil {
		log.Fatalln(err)
	}

	value, err := dm.Get(key)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("key:", key)
	fmt.Println("value:", string(value))
}
