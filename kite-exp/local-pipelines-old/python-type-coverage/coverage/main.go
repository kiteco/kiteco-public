//go:generate go-bindata -o bindata.go templates/...

package main

import (
	"log"

	"github.com/kiteco/kiteco/kite-golib/cmdline"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	cmdline.MustDispatch(measureCmd, showCmd)
}
