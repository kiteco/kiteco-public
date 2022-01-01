// +build !darwin

package process

import (
	"context"
	"path/filepath"

	"github.com/shirou/gopsutil/process"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
)

// Process is a subset of the information provided by gopsutil/process
// we're using it to provide a mocked process list for testing
type Process interface {
	NameWithContext(ctx context.Context) (string, error)
	CmdlineSliceWithContext(ctx context.Context) ([]string, error)
	ExeWithContext(ctx context.Context) (string, error)
}

// List represents a list of processes
type List []Process

// Matching returns a string value for each process in the list where non-empty strings are returned
func (l List) Matching(ctx context.Context, filter func(ctx context.Context, process Process) string) ([]string, error) {
	if len(l) == 0 {
		return []string{}, nil
	}

	var matching []string
	for _, p := range l {
		if v := filter(ctx, p); v != "" {
			matching = append(matching, v)
		}
	}
	return matching, nil
}

// MatchingExeName returns the processes where the name of the executable matches on the the given names
func (l List) MatchingExeName(ctx context.Context, names []string) ([]string, error) {
	return l.Matching(ctx, func(ctx context.Context, process Process) string {
		exe, err := process.ExeWithContext(ctx)
		if err == nil && shared.StringsContain(names, filepath.Base(exe)) {
			return exe
		}
		return ""
	})
}

func list(ctx context.Context) (List, error) {
	list, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	var result []Process
	for _, p := range list {
		result = append(result, p)
	}
	return result, err
}
