package main

import (
	"errors"
	"io/ioutil"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmetrics"
)

type referenceList struct {
	References []*pythonmetrics.ReferenceInfo `json:"references"`
	Filename   string                         `json:"filename"`
}

type ratioValue struct {
	Ratio float64
	Count int64
}

type projectDescription struct {
	ProjectName     string          `json:"project_name"`
	ProjectPath     string          `json:"project_path"`
	VirtualEnvPath  string          `json:"virtual_env_path"`
	SourceSubfolder string          `json:"source_subfolder"`
	ReferenceLists  []referenceList `json:"reference_lists"`
}

func (pd *projectDescription) getSourceFolder() string {
	return pd.ProjectPath + string(os.PathSeparator) + pd.SourceSubfolder
}

func (pd *projectDescription) getLibraryFolder() string {
	return pd.ProjectPath + string(os.PathSeparator) + pd.VirtualEnvPath
}

func (pd *projectDescription) getReferencesForFile(filepath string, samplingRate float64, randomSeed int64) (pythonmetrics.ReferenceMap, error) {
	filename := filepath[len(pd.ProjectPath)+1:]
	for _, refList := range pd.ReferenceLists {
		if refList.Filename == filename {
			return getReferenceMap(refList.References, filepath, samplingRate, randomSeed), nil
		}
	}
	return nil, errors.New("No file named " + filepath)
}

func (pd *projectDescription) getReferenceCount() int {
	var counter int
	for _, refList := range pd.ReferenceLists {
		counter += len(refList.References)
	}
	return counter
}

func getReferenceMap(referenceList []*pythonmetrics.ReferenceInfo, filename string, samplingRate float64, randomSeed int64) pythonmetrics.ReferenceMap {
	content, err := ioutil.ReadFile(filename)
	maybeQuit(err)
	return pythonmetrics.GetReferenceMap(referenceList, content, samplingRate, randomSeed)
}
