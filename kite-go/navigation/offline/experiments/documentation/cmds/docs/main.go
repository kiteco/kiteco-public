package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
)

func main() {
	args := struct {
		Relevant string
		Docs     string
		Index    string
	}{}
	arg.MustParse(&args)
	relevant, err := localpath.NewAbsolute(args.Relevant)
	if err != nil {
		log.Fatal(err)
	}
	dataDir, err := localpath.NewAbsolute(args.Docs)
	if err != nil {
		log.Fatal(err)
	}
	index, err := localpath.NewAbsolute(args.Index)
	if err != nil {
		log.Fatal(err)
	}

	readmePaths, err := findReadmes(relevant)
	if err != nil {
		log.Fatal(err)
	}
	s, err := buildStorage(dataDir, readmePaths)
	if err != nil {
		log.Fatal(err)
	}
	err = s.write(dataDir, index)
	if err != nil {
		log.Fatal(err)
	}
}

func findReadmes(relevant localpath.Absolute) ([]localpath.Absolute, error) {
	var commits map[git.File][]git.File
	file, err := relevant.Open()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &commits)

	kiteco := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "kiteco", "kiteco")
	root, err := localpath.NewAbsolute(kiteco)
	if err != nil {
		return nil, err
	}

	var readmes []localpath.Absolute
	seen := make(map[localpath.Absolute]bool)
	for _, commit := range commits {
		for _, readme := range commit {
			abs := readme.ToLocalFile(root)
			_, err := abs.Lstat()
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if seen[abs] {
				continue
			}
			seen[abs] = true
			readmes = append(readmes, abs)
		}
	}
	return readmes, nil
}

type storage struct {
	index map[localpath.Absolute]localpath.Absolute
	data  map[localpath.Absolute][]byte
}

func buildStorage(dataDir localpath.Absolute, paths []localpath.Absolute) (storage, error) {
	index := make(map[localpath.Absolute]localpath.Absolute)
	data := make(map[localpath.Absolute][]byte)
	for id, path := range paths {
		base := localpath.Relative(fmt.Sprintf("doc%d.py", id))
		dataPath := dataDir.Join(base)

		index[dataPath] = path
		file, err := path.Open()
		if err != nil {
			return storage{}, err
		}
		contents, err := ioutil.ReadAll(file)
		if err != nil {
			return storage{}, err
		}
		data[dataPath] = transform(contents)
	}
	return storage{
		index: index,
		data:  data,
	}, nil
}

func transform(data []byte) []byte {
	parts := []string{`"""`, string(data), `"""`}
	return []byte(strings.Join(parts, "\n"))
}

func (s storage) write(dataDir, indexPath localpath.Absolute) error {
	var storagePermissions os.FileMode = 0600

	indexBytes, err := json.MarshalIndent(s.index, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(string(indexPath), indexBytes, storagePermissions)
	if err != nil {
		return err
	}

	for dataPath, contents := range s.data {
		err := ioutil.WriteFile(string(dataPath), contents, storagePermissions)
		if err != nil {
			return err
		}
	}
	return nil
}
