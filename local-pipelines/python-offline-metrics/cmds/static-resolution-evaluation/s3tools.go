package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

const (
	projectDescriptionFilename = "references-dump.json"
	sourceArchive              = "sources.tar.gz"
)

func fetchS3ProjectList() []string {
	paths, err := fileutil.ListDir(s3ProjectPath)
	maybeQuit(err)
	pSize := len(s3ProjectPath) + 1
	// 2 files per projects, so # of projects = len(paths)/2, the map is used to dedup
	projectList := make(map[string]bool, len(paths)/2)
	for _, path := range paths {
		// Extract project name from S3 path (last part of the path)
		path = path[pSize:]
		i := strings.Index(path, "/")
		path = path[:i]
		projectList[path] = true
	}
	var result []string
	for projectName := range projectList {
		result = append(result, projectName)
	}
	sort.Strings(result)
	return result
}

// contains check is the string s is in the string slice sl
func contains(sl []string, s string) int {
	for i, s2 := range sl {
		if s == s2 {
			return i
		}
	}
	return -1
}

func intersection(s1 []string, s2 []string) []string {
	var result []string
	for _, s := range s1 {
		if contains(s2, s) != -1 {
			result = append(result, s)
		}
	}
	return result
}

func checkProjectFilesArePresent(projectName string) bool {
	path := fileutil.Join(s3ProjectPath, projectName)
	uri, err := awsutil.ValidateURI(path)
	maybeQuit(err)
	bucket := uri.Host
	prefix := uri.Path[1:]
	keys, err := awsutil.S3ListObjects(s3Region, bucket, prefix)
	return contains(keys, fileutil.Join(prefix, projectDescriptionFilename)) != -1 && contains(keys, fileutil.Join(prefix, sourceArchive)) != -1
}

func downloadAndParseProjectDescription(projectName string) (*projectDescription, error) {
	path := fileutil.Join(s3ProjectPath, projectName, projectDescriptionFilename)
	reader, err := awsutil.NewCachedS3Reader(path)
	if err != nil {
		return nil, err
	}
	byteValue, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var project projectDescription
	if err = json.Unmarshal(byteValue, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

func downloadAndExtractFiles(projectName string, project *projectDescription) error {
	target, err := ioutil.TempDir("", projectName)
	if err != nil {
		return err
	}
	path := fileutil.Join(s3ProjectPath, projectName, sourceArchive)
	err = fileutil.ExtractTarGZ(path, target)
	if err != nil {
		return err
	}
	project.ProjectPath = fileutil.Join(target, projectName)
	return nil
}

func downloadAndPrepareProject(projectName string) (*projectDescription, error) {
	if !checkProjectFilesArePresent(projectName) {
		return nil, errors.New("Impossible to process " + projectName + " some files are missing from S3")
	}
	project, err := downloadAndParseProjectDescription(projectName)
	if err != nil {
		return nil, err
	}
	err = downloadAndExtractFiles(projectName, project)
	if err != nil {
		return nil, err
	}
	log.Printf("Project %s extracted in the folder %s", project.ProjectName, project.ProjectPath)

	return project, nil
}
