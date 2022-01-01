package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/bpe"
)

var vocabGenCmd = cmdline.Command{
	Name:     "vocabgen",
	Synopsis: "generate vocab from wordcounts",
	Args: &vocabGenArgs{
		Out:            "vocab.bpe",
		CheckpointsDir: "vocab-checkpoints",
		MaxVocabSize:   20000,
	},
}

type vocabGenArgs struct {
	In             string
	Out            string
	CheckpointsDir string
	MaxVocabSize   int
}

func (args *vocabGenArgs) Handle() error {
	start := time.Now()

	builder := bpe.NewBuilder(true)
	fail(builder.LoadWords(args.In, bpe.LoadOptions{}))

	fail(builder.Merge(bpe.MergeOptions{
		MaxVocabSize:  args.MaxVocabSize,
		Logging:       true,
		Concurrency:   2 * runtime.NumCPU(),
		CheckpointDir: args.CheckpointsDir,
	}))

	f, err := os.Create(args.Out)
	fail(err)
	defer f.Close()

	_, err = builder.WriteTo(f)
	fail(err)

	fmt.Printf("done, took %v\n", time.Since(start))

	return nil
}
