package codesearch

import (
	"bytes"
	"log"
	"os"
	"runtime/pprof"

	"github.com/google/codesearch/index"
	"github.com/google/codesearch/regexp"
)

// SearchOptions ...
type SearchOptions struct {
	L           bool   // L flag - print file names only
	C           bool   // C flag - print count of matches
	N           bool   // N flag - print line numbers
	H           bool   // H flag - do not print file names
	FFlag       string // search only files with names matching this regexp
	IFlag       bool   // case-insensitive search
	VerboseFlag bool   // print extra information
	BruteFlag   bool   // brute force - search all files in index
	CPUProfile  string // write cpu profile to this file
}

// Search ...
func Search(opts SearchOptions, arg string) (*bytes.Buffer, *bytes.Buffer) {
	outBuf := bytes.NewBuffer([]byte{})
	errBuf := bytes.NewBuffer([]byte{})

	g := regexp.Grep{
		Stdout: outBuf,
		Stderr: errBuf,
		L:      opts.L,
		C:      opts.C,
		N:      opts.N,
		H:      opts.H,
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

	pat := "(?m)" + arg
	if opts.IFlag {
		pat = "(?i)" + pat
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		log.Fatal(err)
	}
	g.Regexp = re
	var fre *regexp.Regexp
	if opts.FFlag != "" {
		fre, err = regexp.Compile(opts.FFlag)
		if err != nil {
			log.Fatal(err)
		}
	}
	q := index.RegexpQuery(re.Syntax)
	if opts.VerboseFlag {
		log.Printf("query: %s\n", q)
	}

	ix := index.Open(index.File())
	ix.Verbose = opts.VerboseFlag
	var post []uint32
	if opts.BruteFlag {
		post = ix.PostingQuery(&index.Query{Op: index.QAll})
	} else {
		post = ix.PostingQuery(q)
	}
	if opts.VerboseFlag {
		log.Printf("post query identified %d possible files\n", len(post))
	}

	if fre != nil {
		fnames := make([]uint32, 0, len(post))

		for _, fileid := range post {
			name := ix.Name(fileid)
			if fre.MatchString(name, true, true) < 0 {
				continue
			}
			fnames = append(fnames, fileid)
		}

		if opts.VerboseFlag {
			log.Printf("filename regexp matched %d files\n", len(fnames))
		}
		post = fnames
	}

	for _, fileid := range post {
		name := ix.Name(fileid)
		g.File(name)
	}

	return outBuf, errBuf
}
