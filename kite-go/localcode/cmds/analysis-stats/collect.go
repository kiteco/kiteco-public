package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/readdir"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-go/localfiles/offlineconf"
	"github.com/pkg/errors"
)

// copied from kitelocal/internal/filesystem/manager.go
func walkDir(files map[string]bool, path string) {
	// if !f.isDirAccepted(path) {
	// 	return
	// }

	// throttle the walk
	// f.throttle.Take()

	for _, e := range readdir.List(path) {
		filePath := filepath.Join(path, e.Path)

		var isDir = e.IsDir
		if !e.DTypeEnabled {
			if fi, err := os.Lstat(filePath); err != nil {
				// skip entries we can't process
				log.Printf("walkDir: error %s", err.Error())
				continue
			} else {
				isDir = fi.IsDir()
			}
		}

		if isDir {
			walkDir(files, filePath)
		} else if filepath.Ext(filePath) == ".py" {
			files[filePath] = true
		}
	}
}

func collectLocal(parts ...string) (localcode.BuilderParams, error) {
	root, err := filepath.Abs(parts[0])
	if err != nil {
		return localcode.BuilderParams{}, errors.Wrapf(err, "error computing absolute path for %s: %v", parts[0], err)
	}

	filename, err := filepath.Abs(parts[1])
	if err != nil {
		return localcode.BuilderParams{}, errors.Wrapf(err, "error computing absolute path for %s: %v", parts[1], err)
	}

	var files []*localfiles.File
	fileMap := make(map[string]bool)
	walkDir(fileMap, root)
	for path := range fileMap {
		files = append(files, &localfiles.File{
			Name:          path,
			HashedContent: path,
		})
	}
	if len(files) == 0 {
		return localcode.BuilderParams{}, errors.Errorf("failed to find python files in root directory %s", root)
	}

	return localcode.BuilderParams{
		UserID:     0,
		MachineID:  strings.Replace(root, "/", "_", -1),
		Filename:   filename,
		Files:      files,
		FileGetter: localcode.LocalFileSystem{},
		Local:      true,
	}, nil
}

func collectRemote(parts ...string) (localcode.BuilderParams, error) {
	uid, err := strconv.ParseInt(parts[1], 0, 64)
	if err != nil {
		return localcode.BuilderParams{}, errors.Wrapf(err, "invalid userID %s", parts[1])
	}

	region := parts[0]
	machineID := parts[2]
	filename := parts[3]

	mgr := offlineconf.GetFileManager(region)
	if mgr == nil {
		return localcode.BuilderParams{}, errors.Errorf("could not create FileManager for region %s", region)
	}
	files, err := mgr.List(uid, machineID)
	if err != nil {
		return localcode.BuilderParams{}, errors.Wrapf(err, "could not list files for userID %d and machineID %s", uid, machineID)
	} else if len(files) == 0 {
		return localcode.BuilderParams{}, errors.Errorf("no files listed for userID %d and machineID %s", uid, machineID)
	}

	getter, err := offlineconf.GetFileGetter(region, nil)
	if err != nil {
		return localcode.BuilderParams{}, err
	}

	return localcode.BuilderParams{
		UserID:     uid,
		MachineID:  machineID,
		Filename:   filename,
		Files:      files,
		FileGetter: getter,
		Local:      true,
	}, nil
}

func collect(loc string) (localcode.BuilderParams, error) {
	parts := strings.Split(strings.TrimPrefix(loc, ":"), ":")
	switch len(parts) {
	case 4:
		return collectRemote(parts...)
	case 2:
		return collectLocal(parts...)
	}

	return localcode.BuilderParams{}, errors.Errorf("invalid locator string %s", loc)
}
