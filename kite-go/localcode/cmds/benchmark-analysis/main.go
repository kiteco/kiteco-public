package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/kitelocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	_ "github.com/lib/pq"
)

const (
	corpusURL = "https://github.com/numpy/numpy/blob/8aa121415760cc6839a546c3f84e238d1dfa1aa6/numpy/core/numeric.py"
	// the following limits assume that we are dynamically loading all non-builtin resource data
	maxMemLimit    = 295 * 1024 * 1024 // 295 MB
	buildTimeLimit = 16 * time.Second
)

// - download corpus as zip from github and decompress from memory during build

type zipFS map[string]*zip.File

func newZipFS(r *zip.Reader) zipFS {
	fs := make(zipFS)
	for _, f := range r.File {
		if filepath.Ext(f.Name) != ".py" {
			continue
		}
		// make the path absolute to satisfy the builder
		name := filepath.Join("/", f.Name)
		fs[name] = f
	}
	return fs
}

// Stat implements localcode.FileSystem
func (g zipFS) Stat(path string) (localcode.FileInfo, error) {
	info, ok := g[path]
	if !ok {
		return localcode.FileInfo{}, os.ErrNotExist
	}
	osInfo := info.FileInfo()
	return localcode.FileInfo{
		IsDir: osInfo.IsDir(),
		Size:  osInfo.Size(),
	}, nil
}

// Walk implements localcode.FileSystem
func (g zipFS) Walk(ctx kitectx.Context, path string, walkFn localcode.WalkFunc) error {
	for name := range g {
		isDir := filepath.Dir(name) == path
		if err := walkFn(name, localcode.FileInfo{IsDir: isDir}, nil); err != nil && err != localcode.ErrSkipDir {
			return err
		}
	}
	return nil
}

// Glob implements localcode.FileSystem
func (g zipFS) Glob(dir, pattern string) ([]string, error) {
	var matches []string
	for name := range g {
		if strings.HasPrefix(name, dir) && strings.HasSuffix(name, pattern) {
			matches = append(matches, name)
		}
	}
	return matches, nil
}

// Get implements localcode.FileGetter
func (g zipFS) Get(path string) ([]byte, error) {
	zf, ok := g[path]
	if !ok {
		return nil, errors.Errorf("no file found for path %s", path)
	}

	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ioutil.ReadAll(f)
}

func githubBuilderParams(loc string) (localcode.BuilderParams, error) {
	parts := strings.Split(strings.TrimPrefix(loc, "https://github.com/"), "/")
	// http://github.com/numpy/numpy/blob/8aa121415760cc6839a546c3f84e238d1dfa1aa6/numpy/core/numeric.py
	if len(parts) < 5 || parts[2] != "blob" {
		return localcode.BuilderParams{}, errors.Errorf("expected github blob URL. got %s", loc)
	}
	owner := parts[0]
	repo := parts[1]
	commitish := parts[3]
	path := strings.Join(parts[4:], "/")
	if filepath.Ext(path) != ".py" {
		return localcode.BuilderParams{}, errors.Errorf("expected github blob URL for `.py` file. got %s", loc)
	}

	zipURL := fmt.Sprintf("https://github.com/%s/%s/archive/%s.zip", owner, repo, commitish)
	resp, err := http.Get(zipURL)
	if err != nil {
		return localcode.BuilderParams{}, errors.Wrapf(err, "error fetching archive from URL %s", zipURL)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return localcode.BuilderParams{}, err
	}
	resp.Body.Close()

	zipReader, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return localcode.BuilderParams{}, err
	}

	fs := newZipFS(zipReader)
	// format the start file as a faux-absolute path from the zip archive
	startFile := filepath.Join(fmt.Sprintf("/numpy-%s", commitish), path)

	return localcode.BuilderParams{
		UserID:     1,
		MachineID:  commitish, // arbitrary
		Filename:   startFile,
		FileGetter: fs,
		FileSystem: fs,
		Local:      true,
	}, nil
}

// - test

func loadBuilder() (*pythonbatch.BuilderLoader, error) {
	graph, err := kitelocal.LoadResourceManager(context.Background(), kitelocal.LoadOptions{Blocking: true})
	if err != nil {
		return nil, err
	}

	opts := pythonbatch.DefaultLocalOptions
	// the tracewriter causes extra allocations
	// opts.TraceWriter = os.Stderr

	debug.SetGCPercent(15) // matches the setting in kite-go/client/internal/kitelocal/manager.go

	return &pythonbatch.BuilderLoader{
		Graph:   graph,
		Options: opts,
	}, nil
}

// note that this stops the world, so may have an impact on time-to-build below
func memUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	builder, err := loadBuilder()
	fail(err)
	params, err := githubBuilderParams(corpusURL)
	fail(err)

	// prepopulate the resource manager cache
	_, err = builder.Build(kitectx.Background(), params)
	fail(err)

	runtime.GC()

	var maxMem uint64
	memMonitor := make(chan struct{})
	go func() {
		// log to stderr every 50 milliseconds
		ticker := time.NewTicker(50 * time.Millisecond)
		for {
			select {
			case <-memMonitor:
				ticker.Stop()
				return
			case <-ticker.C:
				mem := memUsage()
				if mem > maxMem {
					maxMem = mem
				}
				// log here to help track down memory allocation
				// log.Printf("[benchmark-analysis] allocated %d bytes\n", mem)
			}
		}
	}()

	results := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := builder.Build(kitectx.Background(), params)
			fail(err)
		}
	})
	close(memMonitor)

	if results.N == 0 {
		fmt.Println("error running benchmark")
		os.Exit(1)
	}

	fmt.Printf("time to build: %d ns\n", results.NsPerOp())
	fmt.Printf("maximum memory allocated: %d B\n", maxMem)

	var failed bool
	if results.NsPerOp() > int64(buildTimeLimit) {
		failed = true
		fmt.Printf("time to build over limit (%d)\n", buildTimeLimit)
	}
	if maxMem > maxMemLimit {
		failed = true
		fmt.Printf("maximum memory allocated over limit (%d)\n", maxMemLimit)
	}

	if failed {
		fmt.Printf("... fail\n")
		os.Exit(1)
	}
	fmt.Printf("... pass\n")
}
