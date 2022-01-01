package localtraining

import (
	"io/ioutil"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/go-errors/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

// WeightedSample ...
type WeightedSample struct {
	LangTag int
	Sample  []int
	Weight  float64
}

func buildSamples(encodedFile []int, contextSize int, toEncode,
	toWeight *lexicalv0.FileEncoder, newMap map[int]bool, weighted bool, filename string) ([]WeightedSample, error) {
	var samples []WeightedSample
	for i := 0; i < len(encodedFile)-contextSize; i++ {
		original := toEncode.PrepareBeforeContext(encodedFile[i:i+contextSize], contextSize, filename)
		var counts int
		var num int
		if !weighted {
			samples = append(samples, WeightedSample{LangTag: toEncode.LangTagForPath(filename), Sample: original, Weight: 1})
			continue
		}
		for _, entry := range original {
			if toEncode.IsLexical(entry) || toEncode.IsEncoderToken(entry) {
				continue
			}
			encodeString := toEncode.IDToString[entry]
			if newMap[entry] {
				weightWords := toWeight.BPE.Encode([]string{encodeString})
				for _, ww := range weightWords {
					e, ok := toWeight.BPE.Entry(ww)
					if !ok {
						return nil, errors.Errorf("new entry not found %s", ww)
					}
					counts += e.Count
					num++
				}
			} else {
				e, ok := toWeight.BPE.Entry(encodeString)
				if !ok {
					return nil, errors.Errorf("old entry not found %s", encodeString)
				}
				counts += e.Count
				num++
			}
		}
		if num != 0 && counts != 0 {
			weight := float64(num) / float64(counts)
			samples = append(samples, WeightedSample{LangTag: toEncode.LangTagForPath(filename), Sample: original, Weight: weight})
		}
	}
	return samples, nil
}

func extractSamples(content []byte, contextSize int, toEncode,
	toWeight *lexicalv0.FileEncoder, newMap map[int]bool, weighted bool, filename string) ([]WeightedSample, error) {
	encoded, err := toEncode.EncodeIdx(content, filename)
	if err != nil {
		return nil, err
	}
	return buildSamples(encoded, contextSize, toEncode, toWeight, newMap, weighted, filename)
}

// RetrieveSamples ...
func RetrieveSamples(files []string, contextSize int, toEncode, toWeight *lexicalv0.FileEncoder,
	newMap map[int]bool, weighted bool) []WeightedSample {

	var gm sync.Mutex
	var completed int64
	var generated []WeightedSample

	var jobs []workerpool.Job
	for _, f := range files {
		fn := f
		jobs = append(jobs, workerpool.Job(func() error {
			defer func() {
				v := atomic.AddInt64(&completed, 1)
				if v%50 == 0 {
					log.Printf("completed %d/%d", completed, len(files))
				}
			}()

			fileContent, err := ioutil.ReadFile(fn)
			if err != nil {
				return nil
			}

			samples, err := extractSamples(fileContent, contextSize, toEncode, toWeight, newMap, weighted, f)
			if err != nil || len(samples) == 0 {
				return nil
			}

			gm.Lock()
			defer gm.Unlock()
			generated = append(generated, samples...)

			return nil
		}))
	}

	pool := workerpool.New(runtime.NumCPU() / 2)
	pool.AddBlocking(jobs)
	err := pool.Wait()
	if err != nil {
		log.Println(err)
	}

	return generated
}

// SelectSamples ...
func SelectSamples(samples []WeightedSample, sampleRate float64, seed int64) []WeightedSample {
	cumulative := make([]float64, 1, len(samples)+1)
	var running float64
	for _, s := range samples {
		running += s.Weight
		cumulative = append(cumulative, running)
	}
	for i := range cumulative {
		cumulative[i] /= running
	}
	ids := sampleWithWeights(cumulative, int(sampleRate*float64(len(samples))), seed)
	selected := make([]WeightedSample, 0, len(ids))
	for _, id := range ids {
		selected = append(selected, samples[id])
	}
	return selected
}

func sampleWithWeights(cum []float64, numSamples int, seed int64) []int {
	if numSamples >= len(cum)-1 {
		all := make([]int, 0, len(cum))
		for i := 0; i < len(cum)-1; i++ {
			all = append(all, i)
		}
		return all
	}
	selectedMap := make(map[int]bool)
	var selected []int
	source := rand.New(rand.NewSource(seed))
	for len(selected) < numSamples {
		r := source.Float64()
		next := oneWeightedSample(cum, r)
		if !selectedMap[next] {
			selectedMap[next] = true
			selected = append(selected, next)
		}
	}
	return selected
}

func oneWeightedSample(cum []float64, r float64) int {
	begin := 0
	end := len(cum) - 1
	for end-begin > 1 {
		middle := (end + begin) / 2
		if r > cum[middle] {
			begin = middle
		} else {
			end = middle
		}
	}
	return begin
}
