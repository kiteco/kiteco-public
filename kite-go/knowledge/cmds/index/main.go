package main

import (
	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/codesearch"
)

func main() {
	args := struct {
		Dirs []string
	}{}
	arg.MustParse(&args)
	opts := codesearch.IndexOptions{
		ResetFlag: true,
	}
	codesearch.Index(opts, args.Dirs...)
}
