package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	_ "golang.org/x/net/trace"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/tfserving"
)

var (
	chartPercentiles = []float64{.25, .5, .75, .9, .95, .99}
)

func computePercentiles(samples []float64) []float64 {
	sort.Float64s(samples)

	var ret []float64
	for _, p := range chartPercentiles {
		idx := int(float64(len(samples)) * p)
		if idx >= len(samples) {
			idx = len(samples) - 1
		}
		ret = append(ret, samples[idx])
	}

	return ret
}

func main() {
	go http.ListenAndServe(":3030", nil)

	var addr string
	var input string
	var window int
	var c, n int
	var print bool
	var slow bool

	flag.StringVar(&input, "input", "", "test file")
	flag.StringVar(&addr, "addr", "localhost:8500", "tensorflow serving address")
	flag.IntVar(&window, "window", 0, "context window")
	flag.IntVar(&c, "c", 1, "num concurrent requesters")
	flag.IntVar(&n, "n", 1, "num requests per requester")
	flag.BoolVar(&print, "print", false, "print results")
	flag.BoolVar(&slow, "slow", false, "slow mode")
	flag.Parse()

	l := lang.FromFilename(input)
	if l == lang.Unknown {
		log.Fatalf("unknown language for input: %s", input)
	}

	buf, err := ioutil.ReadFile(input)
	if err != nil {
		log.Fatalf("error reading input: %s", input)
	}

	opts := lexicalmodels.DefaultModelOptions.WithRemoteModels(addr).RemoteOnly()

	modelName := "all-langs-large"
	encoder, params, config, _ := predict.LoadModelAssets(
		opts.TextMiscGroup.TFServing.ModelPath,
		opts.TextMiscGroup.TFServing.LangGroup,
	)

	enc, err := encoder.EncodeIdx(buf, input)
	if err != nil {
		log.Fatalf("unable to encode file: %+v", err)
	}

	encodedFile := toInt64(enc)

	prefixMask := predict.PrefixIDMask(nil, params.VocabSize)

	if window == 0 {
		window = config.Window
	}

	var completed int64
	var latencies []time.Duration
	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(c)

	time.Sleep(time.Second * 2)

	for i := 0; i < c; i++ {
		go func(wg *sync.WaitGroup) {
			defer wg.Done()

			client, err := tfserving.NewClient(addr, modelName)
			if err != nil {
				log.Fatalln(err)
			}
			defer client.Close()

			windowStart := rand.Intn(len(encodedFile) - window)
			context := encodedFile[windowStart : windowStart+window]
			paddedContext, contextMask := predict.PadContext(context, window, 0)

			for j := 0; j < n; j++ {
				searchStart := time.Now()
				results, probs, err := client.Search(kitectx.Background(), paddedContext, contextMask, prefixMask)
				if err != nil {
					log.Fatalln(err)
				}

				latencies = append(latencies, time.Since(searchStart))
				log.Println(time.Since(searchStart))

				atomic.AddInt64(&completed, 1)

				if slow {
					time.Sleep(time.Second)
				}

				if print {
					log.Printf("results:")
					for _, r := range results {
						log.Println(r)
					}
					log.Printf("probs:")
					for _, r := range probs {
						log.Println(r)
					}
				}
			}
		}(&wg)
	}

	wg.Wait()

	var samples []float64
	for _, l := range latencies {
		samples = append(samples, float64(l))
	}
	percentiles := computePercentiles(samples)
	var durations []time.Duration
	for _, p := range percentiles {
		durations = append(durations, time.Duration(p))
	}

	fmt.Printf("%d concurrent requesters\n", c)
	fmt.Printf("%d requests per requester\n", n)
	fmt.Printf("total time: %v\n", time.Since(start))
	fmt.Printf("requests per second: %.02f\n", float64(completed)/float64(time.Since(start)/time.Second))
	fmt.Println("latency percentiles:")
	for i := 0; i < len(chartPercentiles); i++ {
		fmt.Printf("%.02f: %v\n", chartPercentiles[i], durations[i])
	}
}
func toInt64(a []int) []int64 {
	var b []int64
	if len(a) > 0 {
		b = make([]int64, 0, len(a))
	}
	for _, e := range a {
		b = append(b, int64(e))
	}
	return b
}
