package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type servableModel struct {
	Name     string
	BasePath string
}

func (s *server) servableModelsByName() (map[string]servableModel, error) {
	var models []servableModel
	models = append(models, s.defaultServableModels()...)

	tuned, err := s.tunedServableModels()
	if err != nil {
		return nil, err
	}
	models = append(models, tuned...)

	byName := make(map[string]servableModel)
	for _, model := range models {
		byName[model.Name] = model
	}

	return byName, nil
}

func (s *server) latestTunedModelForRepository(repo string) (servableModel, error) {
	tuned, err := s.tunedServableModels()
	if err != nil {
		return servableModel{}, err
	}

	var forRepo []servableModel
	for _, tm := range tuned {
		if strings.Contains(tm.Name, fmt.Sprintf("-%s-", repo)) {
			forRepo = append(forRepo, tm)
		}
	}

	if len(forRepo) == 0 {
		return servableModel{}, fmt.Errorf("no models found for repo '%s'", repo)
	}

	sort.Slice(forRepo, func(i, j int) bool {
		return forRepo[i].BasePath > forRepo[j].BasePath
	})

	return forRepo[0], nil
}

func (s *server) haveModel(name string) bool {
	fp := filepath.Join(s.tunedModelsDir, name)
	if _, err := os.Stat(fp); err == nil || os.IsExist(err) {
		return true
	}
	return false
}

func (s *server) deleteTunedModel(model string) error {
	if model == "" {
		return nil
	}
	fp := filepath.Join(s.tunedModelsDir, model)
	return os.RemoveAll(fp)
}

func (s *server) defaultServableModels() []servableModel {
	return []servableModel{
		{"default", filepath.Join(s.modelsDir, "all-langs-large", "tfserving")},
	}
}

func (s *server) tunedServableModels() ([]servableModel, error) {
	var models []servableModel

	// check for any tuned models
	modelRoot := s.tunedModelsDir
	fis, err := ioutil.ReadDir(modelRoot)
	if err != nil {
		return nil, err
	}

	for _, fi := range fis {
		m := servableModel{
			Name:     fi.Name(),
			BasePath: filepath.Join(modelRoot, fi.Name(), "tfserving"),
		}
		// Ignore anything thats not a real model (e.g log dir)
		if _, err := os.Stat(m.BasePath); err != nil || os.IsNotExist(err) {
			continue
		}
		models = append(models, m)
	}

	// sort for consistency
	sort.Slice(models, func(i, j int) bool {
		return models[i].Name < models[j].Name
	})

	return models, nil
}

func (s *server) activeServableModel() (servableModel, error) {
	defaultModels := s.defaultServableModels()

	byName, err := s.servableModelsByName()
	if err != nil {
		return servableModel{}, err
	}

	var active servableModel
	for _, dm := range defaultModels {
		versions, err := ioutil.ReadDir(dm.BasePath)
		if err != nil {
			return servableModel{}, err
		}
		var latestVersion string
		for _, version := range versions {
			if version.Name() > latestVersion {
				latestVersion = version.Name()
			}
		}
		if latestVersion == "1" {
			active = dm
		} else if latestVersion == "2" {
			linkedDir, err := os.Readlink(filepath.Join(dm.BasePath, latestVersion))
			if err != nil {
				return servableModel{}, err
			}
			name := filepath.Base(filepath.Join(linkedDir, "..", ".."))
			namedModel, ok := byName[name]
			if !ok {
				return servableModel{}, fmt.Errorf("model name '%s' not found", name)
			}
			active = namedModel
		}
	}

	return active, nil
}

func (s *server) saveActiveModel() error {
	active, err := s.activeServableModel()
	if err != nil {
		return err
	}
	buf, err := json.MarshalIndent(active, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(s.tunedModelsDir, "active.json"), buf, os.ModePerm)
}

func (s *server) loadActiveModel() (servableModel, error) {
	buf, err := ioutil.ReadFile(filepath.Join(s.tunedModelsDir, "active.json"))
	if err != nil {
		return servableModel{}, err
	}

	var active servableModel
	err = json.Unmarshal(buf, &active)
	if err != nil {
		return servableModel{}, err
	}

	// Remove any models that no longer exist
	if _, err := os.Stat(active.BasePath); os.IsNotExist(err) {
		return servableModel{}, fmt.Errorf("active model not found")
	}

	return active, nil
}

func (s *server) linkModel(sm servableModel) error {
	defaultModels := s.defaultServableModels()
	for _, dm := range defaultModels {
		versions, err := ioutil.ReadDir(dm.BasePath)
		if err != nil {
			return err
		}
		for _, version := range versions {
			if version.Name() != "1" {
				err = os.Remove(filepath.Join(dm.BasePath, version.Name()))
				if err != nil {
					return err
				}
			}
		}
		if dm.BasePath == sm.BasePath {
			continue
		}
		err = os.Symlink(
			filepath.Join(sm.BasePath, "1"),
			filepath.Join(dm.BasePath, "2"),
		)
		if err != nil {
			return err
		}
	}
	return nil
}
