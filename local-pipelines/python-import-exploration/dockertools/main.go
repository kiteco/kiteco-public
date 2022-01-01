package main

import (
	"github.com/kiteco/kiteco/kite-golib/cmdline"
)

func main() {
	cmdline.MustDispatch(filesCmd, buildImageCmd, buildImagesCmd, deleteImageCmd, deleteImagesCmd)
}
