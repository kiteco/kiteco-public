package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-server/tunable-models/cmds/internal/api"
	"github.com/mholt/archiver"
)

type tunableRepository struct {
	Name     string
	BasePath string
}

func (s *server) listTunableRepositories() ([]tunableRepository, error) {
	repos, err := ioutil.ReadDir(s.repositoriesDir)
	if err != nil {
		return nil, err
	}

	var tunableRepos []tunableRepository
	for _, repo := range repos {
		tunableRepos = append(tunableRepos, tunableRepository{
			Name:     repo.Name(),
			BasePath: filepath.Join(s.repositoriesDir, repo.Name()),
		})
	}

	return tunableRepos, nil
}

func (s *server) repoPath(name string) string {
	return filepath.Join(s.repositoriesDir, name)
}

func (s *server) haveRepository(name string) bool {
	fp := s.repoPath(name)
	if _, err := os.Stat(fp); err == nil || os.IsExist(err) {
		return true
	}
	return false
}

func (s *server) deleteRepository(name string) error {
	if name == "" {
		return nil
	}
	fp := s.repoPath(name)
	return os.RemoveAll(fp)
}

func (s *server) handleUpload(w http.ResponseWriter, r *http.Request) {
	var resp api.MessageResponse
	tmpDir, err := ioutil.TempDir("", "upload-repo")
	if err != nil {
		resp.Message = fmt.Sprintf("server error: %s", err)
		writeJSON(w, &resp)
		return
	}
	defer os.RemoveAll(tmpDir)

	err = func() error {
		defer r.Body.Close()

		f, err := os.Create(filepath.Join(tmpDir, "targz"))
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(f, r.Body)
		if err != nil {
			return err
		}

		return nil
	}()
	if err != nil {
		resp.Message = fmt.Sprintf("server error: %s", err)
		writeJSON(w, &resp)
		return
	}

	err = archiver.NewTarGz().Unarchive(filepath.Join(tmpDir, "targz"), s.repositoriesDir)
	if err != nil {
		resp.Message = fmt.Sprintf("server error: %s", err)
		writeJSON(w, &resp)
		return
	}

	resp.Message = "repository added succesfully"
	writeJSON(w, &resp)
}
