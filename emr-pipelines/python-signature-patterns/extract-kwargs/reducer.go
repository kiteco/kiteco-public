package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/emr-pipelines/python-signature-patterns/internal/util"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[extrace-kwargs-reducer] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Extracts names of non-explicit keyword args (the entires in **kwargs) from call specs for a function
// Input: all call specs for a specific function, keyed by an anyname for the importgraph node associated with the function
// Output: map from non-explicit keyword arg name to slice of pythoncode.ArgSpecs for the argument.
func main() {
	in := awsutil.NewEMRIterator(os.Stdin)
	out := awsutil.NewEMRWriter(os.Stdout)
	defer out.Close()

	start := time.Now()
	var anyname string
	var kwargs *pythoncode.Kwargs
	var keywords map[string]bool
	var skip bool
	for in.Next() {
		if anyname != in.Key() {
			if kwargs != nil {
				emitKwargs(out, kwargs, anyname)
			}

			// reset
			anyname = in.Key()
			kwargs = nil
			keywords = nil
			skip = false
		}

		if skip {
			continue
		}

		var spec util.CallSpec
		if err := json.Unmarshal(in.Value(), &spec); err != nil {
			log.Fatalf("error unmarshalling specs for %s: %v/n", in.Key(), err)
		}

		if kwargs == nil {
			if spec.NodeArgSpec == nil || spec.NodeArgSpec.Kwarg == "" {
				skip = true
				continue
			}

			keywords = make(map[string]bool)
			for _, arg := range spec.NodeArgSpec.Args {
				if arg.DefaultType != "" {
					keywords[arg.Name] = true
				}
			}

			kwargs = &pythoncode.Kwargs{
				AnyName: spec.AnyName,
				Kwargs:  make(map[string]*pythoncode.Kwarg),
				Name:    spec.NodeArgSpec.Kwarg,
			}
		}

		for _, arg := range spec.Kwargs {
			if keywords[arg.Key] {
				continue
			}

			kw := kwargs.Kwargs[arg.Key]
			if kw == nil {
				kw = &pythoncode.Kwarg{
					Types: make(map[string]int64),
				}
				kwargs.Kwargs[arg.Key] = kw
			}

			kw.Count++
			if arg.Type != "" {
				kw.Types[arg.Type]++
			}
		}
	}

	if kwargs != nil {
		emitKwargs(out, kwargs, anyname)
	}

	if err := in.Err(); err != nil {
		log.Fatalf("error reading from stdin: %v\n", err)
	}
	log.Printf("Done! Took %v\n", time.Since(start))
}

func emitKwargs(out *awsutil.EMRWriter, kwargs *pythoncode.Kwargs, anyname string) {
	if kwargs == nil {
		return
	}

	buf, err := json.Marshal(kwargs)
	if err != nil {
		log.Fatalf("error mashalling kwargs for %s: %v\n", anyname, err)
	}

	if err := out.Emit("Kwargs", buf); err != nil {
		log.Fatalf("error emitting kwargs for %s: %v\n", anyname, err)
	}
}
