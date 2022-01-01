package main

import (
	"path/filepath"

	"github.com/kiteco/kiteco/kite-golib/envutil"
)

const defaultMachine = "default"

var defaultDockerCerts = filepath.Join(envutil.MustGetenv("HOME"), ".docker/machine/machines/default")
