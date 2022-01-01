package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/mitchellh/cli"
)

type terminateregionsCommand struct{}

func (l *terminateregionsCommand) Help() string {
	return "run this command with a release, e.g 'terminateregions release_0'"
}

func (l *terminateregionsCommand) Synopsis() string {
	return "terminate release to all regions"
}

func (l *terminateregionsCommand) Run(args []string) int {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 1 && len(args) != 2 {
		return cli.RunResultHelp
	}

	// read args
	release := args[0]

	// run command in parallel for each region
	sortedRN := sortedRegionNames(Regions)
	resps := make(chan procErr, len(sortedRN))
	wg := sync.WaitGroup{}
	wg.Add(len(sortedRN))
	for _, regionName := range sortedRegionNames(Regions) {
		ui.Info(fmt.Sprintf("starting termination of release %s in region %s", release, regionName))
		go func(r string) {
			defer wg.Done()

			terminateCmd := &terminateCommand{}
			ret := terminateCmd.Run([]string{r, release})

			var err error
			if ret != 0 {
				err = fmt.Errorf("error terminating %s in %s, aborting", release, r)
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
			ui.Error(fmt.Sprintf("failed to terminate region %s:\n\t%v", resp.proc, resp.err))
			failed = true
		}
	}
	if failed {
		return 1
	}

	return 0
}
