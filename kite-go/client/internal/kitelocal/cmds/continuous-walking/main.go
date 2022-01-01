package main

import (
	"flag"
	"log"
	"os/user"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/filesystem"
)

func main() {
	usr, err := user.Current()
	if err != nil {
		log.Fatalln(err)
	}

	var rootdir string
	flag.StringVar(&rootdir, "rootdir", usr.HomeDir, "dir to walk")
	flag.Parse()

	for {
		fs := filesystem.NewManager(filesystem.Options{
			RootDir:   rootdir,
			DutyCycle: 0.15,
			IsFileAccepted: func(path string) bool {
				return filepath.Ext(path) == ".py"
			},
		})

		fs.Initialize(component.InitializerOptions{})
		<-fs.ReadyChan()
	}
}
