package main

import (
	"github.com/kiteco/kiteco/kite-golib/cmdline"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	cmdline.MustDispatch(wordCountCmd, vocabGenCmd)
}
