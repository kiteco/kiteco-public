package execution

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/dgryski/go-spooky"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// TODO(naman) support non-inline?

const (
	cacheSize        = 100
	dockerImage      = "kiteco/answers-sandbox"
	containerWorkdir = "/kite/run/" // see sandbox/Dockerfile
)

// Output represents a code line or output emitted by sandbox/entrypoint.py
type Output struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Data  string `json:"data"`
}

// Block represents an output block emitted by sandbox/entrypoint.py
type Block struct {
	CodeLine *string `json:"code_line,omitempty"`
	Output   *Output `json:"output,omitempty"`
}

type cacheKey struct {
	specKey  string
	codeHash uint64
}

type cacheValue struct {
	blocks []Block
	err    error
}

// Manager manages sandbox state
type Manager struct {
	docker client.APIClient
	cache  *lru.Cache
}

// NewManager makes a new Manager
func NewManager(ctx kitectx.Context) Manager {
	// TODO(naman) reuse the client
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(ctx.Context())

	cache, err := lru.New(cacheSize)
	if err != nil {
		panic(err)
	}
	return Manager{
		docker: cli,
		cache:  cache,
	}
}

// Run runs the given code in a fresh sandbox
func (m Manager) Run(ctx kitectx.Context, spec Spec, code []byte) ([]Block, error) {
	if len(bytes.TrimSpace(code)) == 0 {
		return nil, nil
	}

	key := cacheKey{
		specKey:  spec.HashKey,
		codeHash: spooky.Hash64(code),
	}
	if res, ok := m.cache.Get(key); ok {
		val := res.(cacheValue)
		return val.blocks, val.err
	}

	spec, err := spec.validate()
	if err != nil {
		return nil, errors.Wrapf(err, "invalid execution environment")
	}

	stopTimeout := spec.Timeout * int(time.Millisecond) / int(time.Second)
	// configure docker container
	conf := &container.Config{
		Image:       dockerImage,
		Cmd:         []string{spec.SaveAs},
		StopTimeout: &stopTimeout,
	}
	hostConf := &container.HostConfig{}

	var statusCode int64
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// docker create
	cont, err := m.docker.ContainerCreate(context.Background(), conf, hostConf, nil, "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create new sandbox")
	}
	defer func() {
		err := m.docker.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			log.Println(errors.Wrapf(err, "failed to remove sandbox container %s", cont.ID))
		}
	}()

	err = ctx.WithTimeout(time.Duration(spec.Timeout)*time.Millisecond, func(ctx kitectx.Context) error {
		// docker cp
		var codeTar bytes.Buffer
		codeTarWriter := tar.NewWriter(&codeTar)
		if err := codeTarWriter.WriteHeader(&tar.Header{
			Name:     spec.SaveAs,
			Mode:     0666,
			ModTime:  time.Now(),
			Typeflag: tar.TypeReg,
			Size:     int64(len(code)),
		}); err != nil {
			panic(err)
		}
		if _, err = codeTarWriter.Write(code); err != nil {
			panic(err)
		}
		if err := codeTarWriter.Close(); err != nil {
			panic(err)
		}
		if err := m.docker.CopyToContainer(ctx.Context(), cont.ID, containerWorkdir, &codeTar, types.CopyToContainerOptions{}); err != nil {
			return errors.Wrapf(err, "failed to copy source code to sandbox")
		}

		// docker start
		if err := m.docker.ContainerStart(ctx.Context(), cont.ID, types.ContainerStartOptions{}); err != nil {
			return errors.Wrapf(err, "failed to start sandbox")
		}

		// docker wait
		statusCh, errCh := m.docker.ContainerWait(ctx.Context(), cont.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			return errors.Wrapf(err, "failed to wait for sandbox termination")
		case res := <-statusCh:
			statusCode = res.StatusCode
			if res.Error != nil {
				return errors.Errorf("failed to wait for sandbox termination: %s", res.Error.Message)
			}
		}

		// docker logs
		multiOut, err := m.docker.ContainerLogs(ctx.Context(), cont.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
		if err != nil {
			return errors.Wrapf(err, "failed to collect output from sandbox")
		}
		defer multiOut.Close()
		_, err = stdcopy.StdCopy(&stdout, &stderr, multiOut)
		if err != nil {
			return errors.Wrapf(err, "failed to collect output from sandbox")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// read JSON
	err = nil
	dec := json.NewDecoder(&stdout)
	var blocks []Block
	for dec.More() {
		var block Block
		if err = dec.Decode(&block); err != nil {
			err = errors.Wrapf(err, "error decoding json")
			break
		}
		blocks = append(blocks, block)
	}

	if statusCode != 0 {
		err = errors.Wrapf(err, "sandbox returned non-zero status code %d\nstderr:\n%s\n", statusCode, string(stderr.Bytes()))
	}
	m.cache.Add(key, cacheValue{blocks, err})
	return blocks, err
}
