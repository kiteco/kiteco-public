package codesearch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"

	"github.com/google/codesearch/index"
)

// IndexOptions ...
type IndexOptions struct {
	ListFlag    bool
	ResetFlag   bool
	VerboseFlag bool
	CPUProfile  string
}

// Index ...
func Index(opts IndexOptions, args ...string) {
	if opts.ListFlag {
		ix := index.Open(index.File())
		for _, arg := range ix.Paths() {
			fmt.Printf("%s\n", arg)
		}
		return
	}

	if opts.CPUProfile != "" {
		f, err := os.Create(opts.CPUProfile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if opts.ResetFlag && len(args) == 0 {
		os.Remove(index.File())
		return
	}
	if len(args) == 0 {
		ix := index.Open(index.File())
		for _, arg := range ix.Paths() {
			args = append(args, arg)
		}
	}

	// Translate paths to absolute paths so that we can
	// generate the file list in sorted order.
	for i, arg := range args {
		a, err := filepath.Abs(arg)
		if err != nil {
			log.Printf("%s: %s", arg, err)
			args[i] = ""
			continue
		}
		args[i] = a
	}
	sort.Strings(args)

	for len(args) > 0 && args[0] == "" {
		args = args[1:]
	}

	master := index.File()
	if _, err := os.Stat(master); err != nil {
		// Does not exist.
		opts.ResetFlag = true
	}
	file := master
	if !opts.ResetFlag {
		file += "~"
	}

	ix := index.Create(file)
	ix.Verbose = opts.VerboseFlag
	ix.AddPaths(args)
	for _, arg := range args {
		log.Printf("index %s", arg)
		filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
			if _, elem := filepath.Split(path); elem != "" {
				// Skip various temporary or "hidden" files or directories.
				if elem[0] == '.' || elem[0] == '#' || elem[0] == '~' || elem[len(elem)-1] == '~' {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			if err != nil {
				log.Printf("%s: %s", path, err)
				return nil
			}
			if info != nil && info.Mode()&os.ModeType == 0 {
				ix.AddFile(path)
			}
			return nil
		})
	}
	log.Printf("flush index")
	ix.Flush()

	if !opts.ResetFlag {
		log.Printf("merge %s %s", master, file)
		index.Merge(file+"~", master, file)
		os.Remove(file)
		os.Rename(file+"~", master)
	}
	log.Printf("done")
	return
}
