package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
)

type args struct {
	Source string `arg:"help:path to directory containing files to upload"`
	Dest   string `arg:"help:destination path on s3 to upload files to"`
}

var cmd = cmdline.Command{
	Name:     "dir",
	Synopsis: "Upload contents of a directry to s3",
	Args: &args{
		Source: "../tmp",
		Dest:   "s3://kite-data/python-import-exploration",
	},
}

func (args *args) Handle() error {
	files, err := ioutil.ReadDir(args.Source)
	if err != nil {
		return fmt.Errorf("error reading source dir `%s`: %v", args.Source, err)
	}

	for _, file := range files {
		local := filepath.Join(args.Source, file.Name())
		f, err := os.Open(local)
		if err != nil {
			return fmt.Errorf("error opening `%s`: %v", local, err)
		}

		err = awsutil.S3PutObject(f, join(args.Dest, file.Name()))
		f.Close()
		if err != nil {
			return fmt.Errorf("error writing `%s` to s3: %v", local, err)
		}
	}
	return nil
}

func join(path, part string) string {
	if strings.HasSuffix(path, "/") {
		return path + part
	}
	return path + "/" + part
}

func main() {
	cmdline.MustDispatch(cmd)
}
