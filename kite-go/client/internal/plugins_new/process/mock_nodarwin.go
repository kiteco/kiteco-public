// +build !darwin

package process

import "context"

// NewMockProcess returns a new, mocked process with static values
func NewMockProcess(name string, exe string, cmdline []string) Process {
	return mockProcess{
		name:    name,
		cmdline: cmdline,
		exe:     exe,
	}
}

type mockProcess struct {
	name    string
	exe     string
	cmdline []string
}

func (p mockProcess) NameWithContext(ctx context.Context) (string, error) {
	return p.name, nil
}

func (p mockProcess) CmdlineSliceWithContext(ctx context.Context) ([]string, error) {
	return p.cmdline, nil
}

func (p mockProcess) ExeWithContext(ctx context.Context) (string, error) {
	return p.exe, nil
}
