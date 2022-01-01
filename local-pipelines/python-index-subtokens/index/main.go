package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"sync/atomic"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

var scanOpts = pythonscanner.Options{
	ScanComments: true,
	ScanNewLines: true,
}

func maybeFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

const maxFileBytes = 30000

func main() {
	args := struct {
		Region       string
		In           string
		Out          string
		MaxFiles     int64
		Skip         int
		NumSubTokens int
		DataDir      string
	}{
		Region:       "us-west-1",
		In:           pythoncode.DedupedCodeDumpPath,
		Out:          "index.json",
		MaxFiles:     1000000,
		Skip:         100,
		NumSubTokens: 30000,
	}

	arg.MustParse(&args)

	if args.DataDir != "" {
		awsutil.SetCacheRoot(args.DataDir)
	}

	files, err := aggregator.ListDir(args.In)
	maybeFatal(err)

	sort.Strings(files)

	out, err := os.Create(args.Out)
	maybeFatal(err)
	defer out.Close()

	fmt.Println("starting processing, writing outputs to:", args.Out)

	start := time.Now()

	filesChan := make(chan string)
	go func() {
		for _, f := range files {
			filesChan <- f
		}
		close(filesChan)
	}()

	srcs := make(chan []byte, 1000)

	go func() {
		var used int64

		wp := workerpool.New(4)
		for i := 0; i < 4; i++ {
			job := func() error {
				var count int
				for file := range filesChan {
					rdr, err := fileutil.NewCachedReader(file)
					maybeFatal(err)

					iter := awsutil.NewEMRIterator(rdr)
					for iter.Next() {
						if atomic.LoadInt64(&used) >= args.MaxFiles {
							return nil
						}

						if len(iter.Value()) > maxFileBytes {
							continue
						}

						count++
						if count%args.Skip != 0 {
							continue
						}

						atomic.AddInt64(&used, 1)
						srcs <- iter.Value()
					}

					maybeFatal(iter.Err())
					maybeFatal(rdr.Close())
				}
				return nil
			}
			wp.Add([]workerpool.Job{job})
		}

		wp.Wait()
		wp.Stop()
		close(srcs)
	}()

	toks := make(chan string, 1000000)
	go func() {
		wp := workerpool.New(3)
		for i := 0; i < 3; i++ {
			wp.Add([]workerpool.Job{func() error {
				for src := range srcs {
					subtokens(src, toks)
				}
				return nil
			}})
		}

		wp.Wait()
		wp.Stop()
		close(toks)
	}()

	counts := make(map[string]int, 5000000)
	for tok := range toks {
		counts[tok]++
	}

	fmt.Println("Done collecting subtokens, got", len(counts))

	tcs := make([]tokCount, 0, len(counts))
	for t, c := range counts {
		tcs = append(tcs, tokCount{
			Tok:   t,
			Count: c,
		})
	}

	sort.Slice(tcs, func(i, j int) bool {
		ti, tj := tcs[i], tcs[j]
		if ti.Count == tj.Count {
			// prefer alpha lower
			return ti.Tok < tj.Tok
		}
		// prefer higher count
		return ti.Count > tj.Count
	})

	if len(tcs) > args.NumSubTokens {
		tcs = tcs[:args.NumSubTokens]
	}

	si := make(traindata.SubtokenIndex, len(tcs))
	for _, tc := range tcs {
		si[tc.Tok] = len(si)
	}

	maybeFatal(json.NewEncoder(out).Encode(si))

	fmt.Println("Done! took", time.Since(start))
}

type tokCount struct {
	Tok   string
	Count int
}

func subtokens(buf []byte, out chan string) error {
	words, err := pythonscanner.Lex(buf, scanOpts)
	if err != nil {
		return err
	}

	for _, w := range words {
		if w.Token != pythonscanner.Ident {
			continue
		}

		for _, st := range traindata.SplitNameLiteral(w.Literal) {
			out <- st
		}
	}

	return nil
}
