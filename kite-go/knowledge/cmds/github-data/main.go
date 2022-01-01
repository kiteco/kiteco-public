package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/navigation/offline/pulls"
)

func main() {
	args := struct {
		Owner     string
		Repo      string
		WriteDir  string
		OpenPulls int
		PerPage   int
	}{
		Owner:     "kiteco",
		Repo:      "kiteco",
		OpenPulls: 200,
		PerPage:   25,
	}
	arg.MustParse(&args)

	modes := map[string]int{
		"open": args.OpenPulls,
	}

	for prState, numPulls := range modes {
		writeDir := filepath.Join(args.WriteDir, prState)
		opts := pulls.Options{
			Owner:    args.Owner,
			Repo:     args.Repo,
			WriteDir: writeDir,
			PRState:  prState,
			PerPage:  args.PerPage,
			NumPulls: numPulls,
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
