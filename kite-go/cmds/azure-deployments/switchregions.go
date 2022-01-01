package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/mitchellh/cli"
)

type switchregionsCommand struct{}

func (l *switchregionsCommand) Help() string {
	return "run this command with a release and appgate, e.g 'switchregions release_0 staging'"
}

func (l *switchregionsCommand) Synopsis() string {
	return "switch the appgate to point to the deployment with the specified release in all regions"
}

func (l *switchregionsCommand) Run(args []string) int {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 2 {
		return cli.RunResultHelp
	}

	// read command line args
	release := args[0]
	appgate := args[1]

	// run command for each region
	sortedRN := sortedRegionNames(Regions)
	resps := make(chan procErr, len(sortedRN))
	wg := sync.WaitGroup{}
	wg.Add(len(sortedRN))
	for _, regionName := range sortedRegionNames(Regions) {
		ui.Info(fmt.Sprintf("starting appgate switch for release %s to %s in region %s", release, appgate, regionName))
		go func(r string) {
			defer wg.Done()

			switchCmd := &switchCommand{}
			ret := switchCmd.Run([]string{r, release, appgate})

			var err error
			if ret != 0 {
				err = fmt.Errorf("error switching %s to %s in %s, aborting", release, appgate, r)
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
			ui.Error(fmt.Sprintf("failed to switch region %s:\n\t%v", resp.proc, resp.err))
			failed = true
		}
	}
	if failed {
		return 1
	}

	return 0
}
