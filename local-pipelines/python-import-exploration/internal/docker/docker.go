package docker

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
)

// BuildImage in this machine with the specified name and context,
// Machine may be nil.
func BuildImage(m *Machine, name, dockercontext string) error {
	return BuildWithContext(context.Background(), m, name, dockercontext)
}

// BuildWithContext builds the specified with the provided
// context to carry cancellation signals.
func BuildWithContext(ctx context.Context, m *Machine, name, dockerfilePath string) error {
	var env []string
	if m != nil {
		env = m.env
	}

	dockerfile, err := os.Open(dockerfilePath)
	if err != nil {
		return errors.Wrapf(err, "error opening dockerfile %s", dockerfilePath)
	}
	defer dockerfile.Close()

	err = shellOutContext(ctx, env, dockerfile, "docker", "build", "--force-rm", "--tag", name, "-")
	if err != nil {
		return fmt.Errorf("error building image `%s`: %v", name, err)
	}
	return nil
}

// PushImage with the specified name to the specified registry,
// Machine may be nil.
func PushImage(m *Machine, registry, name string) error {
	var env []string
	if m != nil {
		env = m.env
	}

	tag := path.Join(registry, name)

	err := shellOutEnv(env, "docker", "tag", name, tag)
	if err != nil {
		return fmt.Errorf("error tagging image `%s` for registry `%s`: %v", name, registry, err)
	}

	err = shellOutEnv(env, "docker", "push", tag)
	if err != nil {
		return fmt.Errorf("error pushing image `%s`: %v", name, err)
	}
	return nil
}

// DeleteImage with the specified name,
// this does not delete the image on a remote registry,
// Machine may be nil.
func DeleteImage(m *Machine, name string) error {
	var env []string
	if m != nil {
		env = m.env
	}

	err := shellOutEnv(env, "docker", "image", "rm", "-f", name)
	if err != nil {
		return fmt.Errorf("error deleting image `%s`: %v", name, err)
	}
	return nil
}

// Run a container, Machine may be nil.
func Run(m *Machine, args ...string) error {
	return RunWithContext(context.Background(), m, args...)
}

// RunWithContext runs the specified container and uses the provided
// context to kill the process early or with a timeout.
func RunWithContext(ctx context.Context, m *Machine, args ...string) error {
	var env []string
	if m != nil {
		env = m.env
	}

	fullArgs := append([]string{"run"}, args...)
	err := shellOutContext(ctx, env, nil, "docker", fullArgs...)
	if err != nil {
		return fmt.Errorf("error running container: %v", err)
	}
	return nil
}
