package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons/internal/skeleton"
	"github.com/kiteco/kiteco/kite-golib/serialization"
	yaml "gopkg.in/yaml.v1"
)

// Builds a skeletons data set from raw yaml files
func main() {
	var args struct {
		Skeletons string `arg:"positional,required"`
		Out       string `arg:"positional,required"`
	}
	arg.MustParse(&args)

	builder := skeleton.NewBuilder()
	if err := filepath.Walk(args.Skeletons, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file `%s`: %v", path, err)
		}

		var raw []skeleton.RawNode
		if err := yaml.Unmarshal(buf, &raw); err != nil {
			return fmt.Errorf("error unmarshaling `%s`: %v", path, err)
		}

		fmt.Println("building skeletons for", path)
		start := time.Now()
		if err := builder.Build(raw); err != nil {
			return fmt.Errorf("error building `%s`: %v", path, err)
		}
		fmt.Println("Done! took", time.Since(start))

		return nil
	}); err != nil {
		log.Fatal(err)
	}

	enc, err := serialization.NewEncoder(args.Out)
	if err != nil {
		log.Fatalf("error creating encoder `%s`: %v\n", args.Out, err)
	}
	defer enc.Close()

	if err := enc.Encode(builder); err != nil {
		log.Fatalf("error encoding index: %v\n", err)
	}
}
