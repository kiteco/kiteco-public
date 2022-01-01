package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/mitchellh/cli"
)

type deployregionsCommand struct{}

func (l *deployregionsCommand) Help() string {
	return "run this command with a release, e.g 'deployregions release_0'"
}

func (l *deployregionsCommand) Synopsis() string {
	return "deploy release to all regions"
}

func (l *deployregionsCommand) Run(args []string) int {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if len(args) != 1 {
		return cli.RunResultHelp
	}

	// read args
	release := args[0]

	// run command in parallel for each region
	sortedRN := sortedRegionNames(Regions)
	resps := make(chan procErr, len(sortedRN))
	wg := sync.WaitGroup{}
	wg.Add(len(sortedRN))
	for _, regionName := range sortedRN {
		ui.Info(fmt.Sprintf("starting deploy for release %s on region %s", release, regionName))
		go func(r string) {
			defer wg.Done()

			deployCmd, err := newDeployCommand()
			// add some context to command build error and return if error
			if err != nil {
				err = fmt.Errorf("error building deploy command: %v", err)

				resps <- procErr{
					proc: r,
					err:  err,
				}

				return

			}

			// run deploy command and check error
			ret := deployCmd.Run([]string{r, release})
			if ret != 0 {
				err = fmt.Errorf("error deploying to %s, aborting", r)
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
			ui.Error(fmt.Sprintf("failed to deploy region %s:\n\t%v", resp.proc, resp.err))
			failed = true
		}
	}
	if failed {
		return 1
	}

	return 0
}
