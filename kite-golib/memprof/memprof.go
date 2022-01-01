package memprof

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
)

// A simple wrapper around pprof memory profiling that make it possible to drop in memory
// profiling during debugging by adding just one line:
//
//    memprof.Sample()
//
// will write to /tmp/memprof_<n>.prof where n is incremented on each call
//
// You can also call memprof.SampleTo(path) with a custom path.
//
// To analyze the memory profile:
// $ go tool user-node /tmp/memprof_00000000.prof
//
// Then, to show in web browser:
// (pprof) web
//
// Then, to print as text:
// (pprof) tree
//
// Show the top 5 functions by memory allocations:
// (pprof) top5

func init() {
	path := os.Getenv("KITE_MEMPROF_DIR")
	if path == "" {
		return
	}
	period := time.Second
	if periodStr := os.Getenv("KITE_MEMPROF_PERIOD"); periodStr != "" {
		var err error
		period, err = time.ParseDuration(periodStr)
		if err != nil {
			log.Fatalf("$KITE_MEMPROF_PERIOD was '%s', which is not a valid duration", period)
		}
	}

	ticker := time.NewTicker(period)
	go func() {
		var i int
		for {
			<-ticker.C
			SampleTo(filepath.Join(path, fmt.Sprintf("memprof_%08d.prof", i)))
			i++
		}
	}()
}

// SampleTo writes a heap profile to the given path
func SampleTo(path string) {
	// Open the file
	f, err := os.Create(path)
	if err != nil {
		log.Println("unable to generate heap profile:", err)
		return
	}
	defer f.Close()

	// Write the eahp profile
	err = pprof.WriteHeapProfile(f)
	if err != nil {
		log.Println("unable to generate heap profile:", err)
		return
	}

	log.Println("Wrote heap profile to", path)
}
