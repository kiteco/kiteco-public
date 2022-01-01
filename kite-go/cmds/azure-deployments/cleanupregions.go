package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/mitchellh/cli"
)

type cleanupregionsCommand struct{}

func (l *cleanupregionsCommand) Help() string {
	return "cleanupregions takes no arguments"
}

func (l *cleanupregionsCommand) Synopsis() string {
	return "cleanup releases in all regions"
}

func (l *cleanupregionsCommand) Run(args []string) int {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 0 {
		return cli.RunResultHelp
	}

	// run command in parallel for each region
	sortedRN := sortedRegionNames(Regions)
	resps := make(chan procErr, len(sortedRN))
	wg := sync.WaitGroup{}
	wg.Add(len(sortedRN))
	for _, regionName := range sortedRegionNames(Regions) {
		ui.Info(fmt.Sprintf("starting cleanup of region %s", regionName))
		go func(r string) {
			defer wg.Done()

			cleanupCmd := &cleanupCommand{}
			ret := cleanupCmd.Run([]string{r})

			var err error
			if ret != 0 {
				err = fmt.Errorf("error cleaning up region %s, aborting", r)
			}
			resps <- procErr{
				proc: r,
				err:  err,
			}

		}(regionName)
	}

	// wait for deploys to finish
	go func() {
		wg.Wait()
		close(resps)
	}()

	// check errors
	var failed bool
	for resp := range resps {
		if resp.err != nil {
			ui.Error(fmt.Sprintf("failed to cleanup region %s:\n\t%v", resp.proc, resp.err))
			failed = true
		}
	}
	if failed {
		return 1
	}

	return 0
}
