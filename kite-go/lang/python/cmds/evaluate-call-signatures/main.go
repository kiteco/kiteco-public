//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"github.com/kiteco/kiteco/kite-golib/cmdline"
)

func main() {
	cmdline.MustDispatch(diffCmd, coverageCmd, viewDiffCmd)
}
