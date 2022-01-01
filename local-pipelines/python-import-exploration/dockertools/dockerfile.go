package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/local-pipelines/python-import-exploration/helpers"
)

// see kiteco/python-import-exploration Dockerfile at kite-python/kite_pkgexploration/Dockerfile
var (
	dockerfileTmpl = template.Must(template.New("").Parse(`
FROM kiteco/python-import-exploration

# try 3.7.3 first, then 2.7.16, and if the latter succeeds, set that as the default Python
RUN pip3 install {{.Name}}=={{.Version}} || \
	(pip2 install {{.Name}}=={{.Version}} && pyenv global 2.7.16)

ENTRYPOINT ["/root/.pyenv/shims/python", "-m", "kite.pkgexploration", "{{.Name}}", "{{.Version}}", "{{.Out}}", "{{.Log}}"]
`))
)

func dockerfile(d keytypes.Distribution, w io.Writer) error {
	return dockerfileTmpl.Execute(w, map[string]interface{}{
		"Name":    d.Name,
		"Version": d.Version,
		"Out":     fmt.Sprintf("%s__%s.json", strings.ToLower(d.Name), d.Version), // lowercase package to make consistent with image name
		"Log":     fmt.Sprintf("%s__%s.log", strings.ToLower(d.Name), d.Version),  // lowercase package to make consistent with image name
	})
}

type filesArgs struct {
	Force bool   `arg:"-f" help:"overwrite preexisting dockerfiles"`
	In    string `arg:"positional,required,help:input file containing list of packages for import explorarion"`
	Out   string `arg:"positional,required,help:output directory to write dockerfiles to"`
}

var filesCmd = cmdline.Command{
	Name:     "files",
	Synopsis: "build a set of dockerfiles for import exploration of a set of packages",
	Args:     &filesArgs{},
}

func (args *filesArgs) Handle() error {
	buf, err := ioutil.ReadFile(args.In)
	if err != nil {
		return errors.Wrapf(err, "error reading %s", args.In)
	}

	// if we need to cleanup partial results
	var files []string

	if err := os.MkdirAll(args.Out, os.ModePerm); err != nil {
		return errors.Wrapf(err, "error making output dir %s", args.Out)
	}

	err = helpers.ReadLinesWithComments(string(buf), func(distStr, comment string) error {
		if distStr == "" {
			return nil
		}
		d, err := keytypes.ParseDistribution(distStr)
		if err != nil {
			log.Printf("could not parse distribution string %s", distStr)
			return nil
		}

		path := filepath.Join(args.Out, strings.ToLower(fmt.Sprintf("%s__%s", d.Name, d.Version)))
		if !args.Force {
			if _, err := os.Stat(path); err == nil {
				return nil // the Dockerfile already exists, and we're not in --force
			}
		}

		f, err := os.Create(path)
		if err != nil {
			return errors.Wrapf(err, "error creating file %s", path)
		}
		files = append(files, path)

		err = dockerfile(d, f)
		f.Close()
		if err != nil {
			return errors.Wrapf(err, "error creating dockerfile for distribution %s", d)
		}
		return nil
	})
	if err != nil {
		var errs []error
		for _, file := range files {
			if cerr := os.Remove(file); cerr != nil {
				errs = append(errs, cerr)
			}
		}

		if len(errs) > 0 {
			var strs []string
			for _, err := range errs {
				strs = append(strs, err.Error())
			}
			log.Printf("could not clean up: %s", strings.Join(strs, ","))
		}
		log.Fatalln(err)
	}

	return nil
}
