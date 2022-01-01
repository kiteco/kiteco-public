package sandbox

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

var (
	// Random stream for the random container name generator
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	// The docker client
	dockerClient *docker.Client
	// The docker host
	dockerHost string
)

func init() {
	// Note that docker.NewClientFromEnv does not open any network so it is safe to do
	// this in init(). It can fail if the standard docker environment variables point
	// to x509 certificates that do not exist, or are not valid x509 certificates, or
	// if the value of DOCKER_HOST fails to parse as a URL
	var err error
	dockerClient, err = docker.NewClientFromEnv()
	if err != nil {
		log.Fatalln(err)
	}

	dockerHost, err = getDockerHost()
	if err != nil {
		log.Fatalln(err)
	}
}

func getDockerHost() (string, error) {
	// Unfortunately we must separately deduce the host for the docker daemon -- the
	// docker client we use does this same logic internally but does not expose the
	// results in any way!
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		return "127.0.0.1", nil
	}

	uri, err := url.Parse(host)
	if err != nil {
		return "", err
	}

	h := strings.Split(uri.Host, ":")[0]
	if h == "" {
		return "", fmt.Errorf("DOCKER_HOST had no host: %s", host)
	}

	return h, nil
}

func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rng.Intn(len(letters))]
	}
	return string(b)
}

// Container is a thin wrapper around a docker container in which a process is currently running,
// together with a subprocess in which "docker run" is executing that is used to communicate with
// the container. In the future we should switch to using the docker API directly rather than
// communicating via the "docker" executable. Note that Container implements the Process interface.
type Container struct {
	// reference to the docker client
	client *docker.Client
	// the underlying docker container
	container *docker.Container
	// will receive a message upon timeout (started in StartContainer)
	timeout <-chan time.Time
	// internal container port
	internalPort int
	// host for the docker daemon
	host string
	// options with which the container was created
	opts *ProcessOptions
}

// construct a tar archive out of a map from paths to buffers
func tarMap(wr io.Writer, files map[string][]byte) error {
	w := tar.NewWriter(wr)
	defer w.Close()
	now := time.Now()
	for path, contents := range files {
		err := w.WriteHeader(&tar.Header{
			Name:       path,
			Mode:       0777,
			Typeflag:   tar.TypeReg,
			Size:       int64(len(contents)),
			ModTime:    now, // some scripts complain with empty timestamps so set to now
			AccessTime: now,
			ChangeTime: now,
		})
		if err != nil {
			return err
		}
		_, err = w.Write(contents)
		if err != nil {
			return err
		}
	}
	return nil
}

// construct a map from paths to buffers from a tar archive
func untarMap(rd io.Reader) (map[string][]byte, error) {
	files := make(map[string][]byte)
	r := tar.NewReader(rd)
	for {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		// Ignore everything other than regular files
		if !h.FileInfo().Mode().IsRegular() {
			continue
		}
		files[h.Name], err = ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

// StartContainer starts a new container
func StartContainer(image string, opts *ProcessOptions) (*Container, error) {
	if opts.Command == "" {
		return nil, fmt.Errorf("ProcessOptions.Command must not be empty")
	}
	if opts.WorkingDir == "" {
		return nil, fmt.Errorf("ProcessOptions.WorkingDir must not be empty")
	}
	if len(opts.Files) > 0 && opts.FileReader != nil {
		return nil, fmt.Errorf("ProcessOptions.Files and ProcessOptions.FileReader must not be used together")
	}

	c := Container{
		client: dockerClient,
		host:   dockerHost,
		opts:   opts,
	}

	var hostConfig docker.HostConfig
	containerConfig := docker.Config{
		Cmd:          append([]string{opts.Command}, opts.Args...),
		Image:        image,
		ExposedPorts: make(map[docker.Port]struct{}),
		WorkingDir:   opts.WorkingDir,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	for name, value := range opts.EnvironmentVariables {
		containerConfig.Env = append(containerConfig.Env, name+"="+value)
	}

	if opts.Port != 0 {
		port := docker.Port(fmt.Sprintf("%d/tcp", opts.Port))
		// tell docker to select an unused port for each ExposedPort
		hostConfig.PublishAllPorts = true
		containerConfig.ExposedPorts[port] = struct{}{}
		c.internalPort = opts.Port
	}

	// Create the container
	log.Println("Creating container...")
	var err error
	c.container, err = c.client.CreateContainer(docker.CreateContainerOptions{
		Config:     &containerConfig,
		HostConfig: &hostConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("error during CreateContainer: %v", err)
	}
	log.Println("Created container with ID", c.container.ID)

	// Upload files into container
	fileRd := opts.FileReader
	if len(opts.Files) > 0 {
		// Make absolute paths for the files
		absFiles := make(map[string][]byte)
		for pth, data := range opts.Files {
			if !path.IsAbs(pth) {
				pth = path.Join(opts.WorkingDir, pth)
			}
			absFiles[pth] = data
		}

		// Create a tar archive
		var buf bytes.Buffer
		err := tarMap(&buf, absFiles)
		if err != nil {
			return nil, fmt.Errorf("error creating tar archive of files: %v", err)
		}
		fileRd = &buf
	}
	if fileRd != nil {
		// Upload the archive to the container
		err = c.client.UploadToContainer(c.container.ID, docker.UploadToContainerOptions{
			InputStream: fileRd,
			Path:        "/",
		})
		if err != nil {
			return nil, fmt.Errorf("error during UploadToContainer: %v", err)
		}
	}

	// Run the container
	log.Println("Starting container...")
	err = c.client.StartContainer(c.container.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("error during StartContainer: %v", err)
	}
	log.Println("Container started successfully")

	// Fetch updated details for the container
	log.Println("Inspecting container...")
	details, err := c.client.InspectContainer(c.container.ID)
	if err != nil {
		return nil, fmt.Errorf("error during InspectContainer: %v", err)
	}
	c.container = details
	log.Println("Container inspected successfully")

	// Start timer
	c.timeout = opts.Limits.makeTimeoutChannel()

	return &c, nil
}

func (c *Container) output() ([]byte, []byte, error) {
	var stdout, stderr bytes.Buffer
	err := c.client.Logs(docker.LogsOptions{
		Container:    c.container.ID,
		OutputStream: &stdout,
		ErrorStream:  &stderr,
		Stdout:       true,
		Stderr:       true,
	})
	if err != nil {
		log.Println("error when fetching output from docker with Logs():", err)
	}
	return stdout.Bytes(), stderr.Bytes(), err
}

// Wait blocks the program exits, then returns the output generated
func (c *Container) Wait() ([]byte, []byte, error) {
	rescind := make(chan error)
	return c.RescindableWait(rescind)
}

// RescindableWait blocks until either the program exits or any message is received on the given channel.
// If a message on the rescind channel terminates the program's execution then that object is returned
// as the third output parameter of this function.
func (c *Container) RescindableWait(rescind chan error) ([]byte, []byte, error) {
	// Wait for all streams to be closed
	type waitResult struct {
		exitcode int
		err      error
	}
	wait := make(chan waitResult, 1)
	go func() {
		exitcode, err := c.client.WaitContainer(c.container.ID)
		wait <- waitResult{exitcode, err}
	}()

	// Wait for either command to complete or a limit to be exceeded
	select {
	case <-c.timeout:
		log.Println("Killing container after timeout expired")
		stdout, stderr, _ := c.kill()
		return stdout, stderr, &TimeLimitExceeded{}

	case err := <-rescind:
		log.Printf("Killing container after rescind: %v\n", err)
		stdout, stderr, _ := c.kill()
		return stdout, stderr, err

	case result := <-wait:
		log.Printf("Container exited with status %d, error %v\n", result.exitcode, result.err)
		if result.err == nil && result.exitcode != 0 {
			result.err = fmt.Errorf("process exited with status %d", result.exitcode)
		}
		return c.output()
	}
}

// Cancel terminates the underlying subprocess and returns output generated so far
func (c *Container) Cancel() ([]byte, []byte) {
	log.Println("Killing container after action finished, regardless of error")
	stdout, stderr, _ := c.kill()
	return stdout, stderr
}

// Endpoint returns the host:port that can be used to communicate with the program. This
// may differ from the port passed to Start()
func (c *Container) Endpoint() (string, error) {
	port := docker.Port(fmt.Sprintf("%d/tcp", c.internalPort))
	binding, exists := c.container.NetworkSettings.Ports[port]
	if !exists {
		return "", fmt.Errorf("no binding for %s (%v)", port, c.container.NetworkSettings.Ports)
	}
	if len(binding) != 1 {
		return "", fmt.Errorf("expected one binding but found: %v", binding)
	}
	endpoint := c.host + ":" + binding[0].HostPort
	return endpoint, nil
}

// kill terminates the underlying subprocess and returns output generated so far, but does no cleanup
func (c *Container) kill() ([]byte, []byte, error) {
	stdout, stderr, err := c.output()

	errKill := c.client.KillContainer(docker.KillContainerOptions{
		ID: c.container.ID,
	})
	if errKill != nil {
		log.Printf("Error sending SIGKILL to docker container: %v\n", errKill)
	}
	log.Println("Container killed successfully")

	return stdout, stderr, err
}

// Files gets the files in the working dir of the process
func (c *Container) Files() (map[string][]byte, error) {
	// Download files from container
	var buf bytes.Buffer
	err := c.client.DownloadFromContainer(c.container.ID, docker.DownloadFromContainerOptions{
		OutputStream: &buf,
		Path:         c.opts.WorkingDir,
	})
	if err != nil {
		return nil, err
	}

	// Untar the buffer
	absFiles, err := untarMap(&buf)
	if err != nil {
		return nil, err
	}

	// Compute relative paths
	files := make(map[string][]byte)
	for path, data := range absFiles {
		relPath, err := filepath.Rel(c.opts.WorkingDir, "/"+path)
		if err != nil {
			return nil, err
		}
		files[relPath] = data
	}
	return files, nil
}

// Cleanup deletes the underlying docker container
func (c *Container) Cleanup() {
	log.Printf("Cleaning up docker container %s", c.container.ID)
	err := c.client.RemoveContainer(docker.RemoveContainerOptions{
		ID:            c.container.ID,
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		log.Printf("Error deleting docker container %s: %v. Ignoring.", c.container.ID, err)
	}
}

// --

// ContinuousContainer wraps a docker container that runs continuously until a termination command
// is evoked. The image used to create the container must contain the command that execute the program
// that will run continuously. An example of a program of such is can be a script that listens to
// stdin and processes the data it receives on stdin. The user must provide the input and output streams
// that are attached to the container.
type ContinuousContainer struct {
	// Reference to the docker client
	client *docker.Client
	// The underlying docker container
	container *docker.Container

	// Host for the docker daemon
	host string
}

// StartContinuousContainer starts a new ContinuousContainer with the given docker image and
// the given input and output stream to be attached with the container.
func StartContinuousContainer(image string, input io.ReadCloser, output io.WriteCloser) (*ContinuousContainer, error) {
	c := ContinuousContainer{
		client: dockerClient,
		host:   dockerHost,
	}

	var hostConfig docker.HostConfig
	containerConfig := docker.Config{
		Image:        image,
		StdinOnce:    true,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
	}

	// Create the container
	log.Println("Creating container...")
	var err error
	c.container, err = c.client.CreateContainer(docker.CreateContainerOptions{
		Config:     &containerConfig,
		HostConfig: &hostConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("error during CreateContainer in StartContinuousContainer: %v", err)
	}
	log.Println("Created container with ID", c.container.ID)

	// Run the container
	// Attach the input and output streams to the container
	attachOpts := docker.AttachToContainerOptions{
		Container:    c.container.ID,
		InputStream:  input,
		OutputStream: output,
		Stdin:        true,
		Stdout:       true,
		Stream:       true,
	}
	go func() {
		if err := c.client.AttachToContainer(attachOpts); err != nil {
			log.Println("attaching IO to container caused error:", err)
		}
	}()

	log.Println("Starting container...")
	err = c.client.StartContainer(c.container.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("error during StartContainer in StartContinuousContainer: %v", err)
	}
	log.Println("Container started successfully")

	return &c, nil
}

// Kill terminates the associated container.
func (c *ContinuousContainer) Kill() error {
	err := c.client.KillContainer(docker.KillContainerOptions{
		ID: c.container.ID,
	})
	if err != nil {
		return fmt.Errorf("error sending SIGKILL to docker container for a ContinuousContainer: %v", err)
	}
	return nil
}

// Errors returns the content in the container's stderr.
func (c *ContinuousContainer) Errors() string {
	var stderr bytes.Buffer
	err := c.client.Logs(docker.LogsOptions{
		Container:   c.container.ID,
		ErrorStream: &stderr,
		Stderr:      true,
	})
	if err != nil {
		log.Println("error when fetching output from docker with Logs():", err)
	}
	return string(stderr.Bytes())
}
