package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func shellOut(name string, args ...string) error {
	return shellOutEnv(nil, name, args...)
}

func shellOutEnv(env []string, name string, args ...string) error {
	return shellOutContext(context.Background(), env, nil, name, args...)
}

func shellOutContext(ctx context.Context, env []string, stdin io.Reader, name string, args ...string) error {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = stdin
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running cmd `%s %s`: stdout:\n%s\nstderr:\n%s\nerr: %v",
			name, strings.Join(args, " "), stdout.String(), stderr.String(), err)
	}
	return nil
}
