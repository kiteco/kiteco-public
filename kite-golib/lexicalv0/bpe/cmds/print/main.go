package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/bpe"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	args := struct {
		Vocab      string
		WordCounts string
		Out        string
	}{}
	arg.MustParse(&args)

	if args.Vocab == "" && args.WordCounts == "" {
		fail(errors.New("must provide either --vocab or --wordcounts or both"))
	}

	out := os.Stdout
	if args.Out != "" {
		f, err := os.Create(args.Out)
		fail(err)
		defer f.Close()
		out = f
	}

	if args.WordCounts != "" {
		f, err := fileutil.NewCachedReader(args.WordCounts)
		fail(err)
		defer f.Close()

		var counts []bpe.BuilderWordCount
		fail(json.NewDecoder(f).Decode(&counts))

		sort.Slice(counts, func(i, j int) bool {
			return counts[i].Count > counts[j].Count
		})

		fmt.Fprintf(out, "Word counts:\n")
		for _, wc := range counts {
			s := string(wc.Word)
			fmt.Fprintf(out, "'%s'   % x   %q   %d\n", s, s, s, wc.Count)
		}
	}

	if args.Vocab != "" {
		enc, err := bpe.NewEncoder(args.Vocab)
		fail(err)

		entries := enc.Entries()
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Count > entries[j].Count
		})

		fmt.Fprintf(out, "Vocab entries:\n")
		for _, e := range entries {
			fmt.Fprintf(out, "'%v'   % x   %q   %d\n", e.BytePair, e.BytePair, e.BytePair, e.Count)
		}
		fmt.Fprintf(out, "num vocab entries: %d\n", len(entries))
	}
}
