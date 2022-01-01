package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/navigation/offline/pulls"
	"github.com/kiteco/kiteco/kite-go/navigation/offline/validation"
)

func main() {
	args := struct {
		ReposPath string
		WriteDir  string
		OpenPulls int
		PerPage   int
	}{
		OpenPulls: 200,
		PerPage:   25,
	}
	arg.MustParse(&args)

	repos, err := validation.ReadRepos(args.ReposPath)
	if err != nil {
		log.Fatal(err)
	}
	modes := []mode{
		mode{
			prState:  "open",
			numPulls: args.OpenPulls,
		},
	}

	log.Println("Extracting repos:")
	for _, repo := range repos {
		log.Printf("%s/%s\n", repo.Owner, repo.Name)
	}

	for _, repo := range repos {
		for _, mode := range modes {
			writeDir := filepath.Join(args.WriteDir, repo.Owner, repo.Name, mode.prState)
			opts := pulls.Options{
				Owner:    repo.Owner,
				Repo:     repo.Name,
				WriteDir: writeDir,
				PRState:  mode.prState,
				PerPage:  args.PerPage,
				NumPulls: mode.numPulls,
			}
			err := os.MkdirAll(writeDir, 0700)
			if err != nil {
				log.Fatal(err)
			}
			err = pulls.ExtractPulls(opts)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

type mode struct {
	prState  string
	numPulls int
}
