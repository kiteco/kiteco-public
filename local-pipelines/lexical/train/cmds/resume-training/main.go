package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

func main() {
	args := struct {
		ResumeFrom   string
		ConfigInput  string
		ConfigOutput string
		Output       string
		Steps        int
	}{}

	arg.MustParse(&args)

	inConfig, err := predict.NewHParams(args.ConfigInput)
	if err != nil {
		log.Fatalln(err)
	}

	modelConfig, err := predict.NewHParams(fileutil.Join(args.ResumeFrom, "config.json"))
	if err != nil {
		log.Fatalln(err)
	}

	// Doesn't affect model compatability, use the "newer" value
	modelConfig.NumPredictionSlots = inConfig.NumPredictionSlots

	if inConfig != modelConfig {
		log.Fatalf("model configurations are incompatible:\ncurrent: %+v\nmodel: %+v",
			inConfig, modelConfig)
	}

	buf, err := json.Marshal(modelConfig)
	if err != nil {
		log.Fatalln(err)
	}

	err = ioutil.WriteFile(args.ConfigOutput, buf, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}

	steps := args.Steps
	if args.Steps == 0 {
		parts := strings.Split(filepath.Base(args.ResumeFrom), "_")
		var err error
		steps, err = strconv.Atoi(parts[14])
		if err != nil {
			log.Fatalln(err)
		}
	}

	listing, err := fileutil.ListDir(args.ResumeFrom)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range listing {
		src := item
		dst := filepath.Join(args.Output, strings.TrimPrefix(item, args.ResumeFrom))
		fmt.Printf("%s -> %s\n", src, dst)
		func() {
			r, err := fileutil.NewReader(src)
			if err != nil {
				log.Fatalln(err)
			}
			defer r.Close()

			dir := fileutil.Dir(dst)
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				log.Fatalln(err)
			}

			d, err := os.Create(dst)
			if err != nil {
				log.Fatalln(err)
			}
			defer d.Close()

			_, err = io.Copy(d, r)
			if err != nil {
				log.Fatalln(err)
			}
		}()
	}

	f, err := os.Create(filepath.Join(args.Output, "steps"))
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	fmt.Fprintf(f, "%d", steps)
}
