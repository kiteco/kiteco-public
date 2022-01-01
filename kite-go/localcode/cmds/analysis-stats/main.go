package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/distranking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/pkg/errors"
)

type buildData struct {
	artifact *pythonlocal.SymbolIndex
	resolved *pythonanalyzer.ResolvedAST
	times    struct {
		resolve time.Duration
		build   time.Duration
	}
}

func build(builder *pythonbatch.BuilderLoader, params localcode.BuilderParams) (buildData, error) {
	var b buildData

	ts := time.Now()
	buildResult, err := builder.Build(kitectx.Background(), params)
	if err != nil {
		return b, err
	}
	b.artifact = buildResult.LocalArtifact.(*pythonlocal.SymbolIndex)
	b.times.build = time.Since(ts)

	var contents []byte
	var found bool
	for _, f := range params.Files {
		if f.Name == params.Filename {
			contents, err = params.FileGetter.Get(f.HashedContent)
			if err != nil {
				return b, errors.Wrapf(err, "error getting file %s with hash %s", f.Name, f.HashedContent)
			}
			found = true
			break
		}
	}
	if !found {
		return b, errors.Errorf("file %s not found", params.Filename)
	}

	ts = time.Now()
	mod, err := pythonparser.Parse(kitectx.Background(), contents, pythonparser.Options{
		Approximate: true,
		ScanOptions: pythonscanner.Options{Label: params.Filename},
	})
	if mod == nil {
		return b, errors.Wrapf(err, "error parsing file %s", params.Filename)
	}
	resolved, err := pythonanalyzer.Resolve(kitectx.Background(), pythonanalyzer.Models{
		Importer: pythonstatic.Importer{
			Path:        params.Filename,
			PythonPaths: b.artifact.PythonPaths,
			Global:      builder.Graph,
			Local:       b.artifact.SourceTree,
		},
	}, mod, pythonanalyzer.Options{
		User:    params.UserID,
		Machine: params.MachineID,
		Path:    params.Filename,
		Trace:   builder.Options.TraceWriter,
	})
	if err != nil {
		return b, errors.Wrapf(err, "error resolving AST for file %s", params.Filename)
	}
	b.resolved = resolved
	b.times.resolve = time.Since(ts)

	return b, nil
}

func mustLoadBuilder() *pythonbatch.BuilderLoader {
	// simulate what Kite Local does
	resourceOpts := pythonresource.DefaultLocalOptions
	resourceOpts.Dists = make([]keytypes.Distribution, 0, 1) // since nil -> load all distributions
	for dist, ranking := range distranking.DefaultRanking {
		if ranking < 50 {
			resourceOpts.Dists = append(resourceOpts.Dists, dist)
		}
	}

	graph, errc := pythonresource.NewManager(resourceOpts)
	if err := <-errc; err != nil {
		panic(err)
	}

	opts := pythonbatch.DefaultLocalOptions

	return &pythonbatch.BuilderLoader{
		Graph:   graph,
		Options: opts,
	}
}

func use(stuff ...interface{}) {
	for _, x := range stuff {
		_ = x
	}
}

func memUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: ./benchmark-analysis [root_py_dir:working_file.py | region:uid:mid:filename] ...")
		os.Exit(1)
	}

	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatalln("nothing to build")
	}

	// for blocking before continuing
	reader := bufio.NewReader(os.Stdin)

	// for pprof debugging
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	fmt.Printf("loading builder...\n")
	builder := mustLoadBuilder()
	fmt.Printf("builder loaded; allocated %d bytes\n", memUsage())
	debug.SetGCPercent(15)

	for _, arg := range args {
		runtime.GC()
		fmt.Printf("building %s\n", arg)

		// monitor memory usage
		memMonitor := make(chan struct{})
		go func() {
			// print to stdout every second
			fmtTicker := time.NewTicker(time.Second)
			// log to stderr every 50 milliseconds
			logTicker := time.NewTicker(50 * time.Millisecond)
			for {
				select {
				case <-memMonitor:
					fmtTicker.Stop()
					logTicker.Stop()
					return
				case <-fmtTicker.C:
					fmt.Printf("... allocated %d bytes\n", memUsage())
				case <-logTicker.C:
					log.Printf("[benchmark-analysis] allocated %d bytes\n", memUsage())
				}
			}
		}()

		fmt.Println("... collecting")
		params, err := collect(arg)
		if err != nil {
			log.Printf("[ERROR] %s\n", err)
			continue
		}

		fmt.Println("... building")
		dat, err := build(builder, params)
		if err != nil {
			log.Printf("[ERROR] %s\n", err)
			continue
		}

		time.Sleep(time.Second)
		close(memMonitor)

		// give the user time to generate a pprof profile
		fmt.Printf("... built %s; enter newline to continue", arg)
		reader.ReadString('\n')

		// to prevent dat from being collected
		log.Printf("... dat = %v", dat)
	}
}
