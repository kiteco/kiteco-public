package localcode

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang"
)

// StandaloneClient allows stand-alone interactions with local-code-worker
// e.g retrieving and loading artifacts.
type StandaloneClient struct {
	workers   *workerGroup
	artifacts *artifactClient
}

// NewStandaloneClient creates a client connected to the provided host:port.
func NewStandaloneClient(hostPort string) (*StandaloneClient, error) {
	workers, err := newWorkerGroupHostPort(hostPort)
	if err != nil {
		return nil, err
	}

	return &StandaloneClient{
		workers:   workers,
		artifacts: newArtifactClient(workers),
	}, nil
}

// FindArtifact will query the Worker for the latest artifact for the provided user, machine and filename
func (s *StandaloneClient) FindArtifact(uid int64, machine, filename string) (interface{}, error) {
	artifact, err := s.artifacts.findArtifact(userMachineFile{uid, machine, filename})
	if err != nil {
		return nil, err
	}

	lang := lang.FromFilename(filename)
	loader, ok := getLoader(lang)
	if !ok {
		return nil, fmt.Errorf("no loader for %s found", lang.Name())
	}

	getter := newArtifactGetter(artifact, s.artifacts)
	obj, err := loader(getter)
	return obj, err
}
