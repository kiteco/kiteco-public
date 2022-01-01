package main

import (
	"fmt"
	"log"

	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/spf13/cobra"
)

func init() {
	log.SetPrefix("[rundb] ")
}

func createCmd() *cobra.Command {
	// TODO: can have extra argument for run name
	// TODO: can have command-line arguments for params
	return &cobra.Command{
		Use:   "create RUNDB_DIR PIPELINE",
		Short: "create a RunDB entry within the given directory",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			dir := args[0]
			pipe := args[1]

			rdb, err := rundb.NewRunDB(dir)
			fail(err)

			ri := rundb.NewRunInfo(rdb, pipe, "")
			ri.SetStatus(rundb.StatusStarted)
			fail(ri.Save())

			fmt.Println(ri.S3Path())
		},
	}
}

func addArtifactCmd() *cobra.Command {
	var recursive *bool

	cmd := cobra.Command{
		Use:   "add-artifact RUN_PATH LOCAL_PATH REMOTE_REL_PATH",
		Short: "uploads a local file to the given remote path relative to the path of the run",
		Args:  cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			runPath := args[0]
			localPath := args[1]
			remoteRelPath := args[2]

			// assert that RUN_PATH points to an actual rundb entry
			_, err := rundb.NewRunInfoFromPath(runPath)
			fail(err)

			remotePath := fmt.Sprintf("%s/%s", runPath, remoteRelPath)

			fail(upload(localPath, remotePath, *recursive))
		},
	}

	recursive = cmd.Flags().Bool("recursive", false, "add a directory recursively")

	return &cmd
}

func waitArtifactCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "wait-artifact RUN_PATH REMOTE_REL_PATH",
		Short: "uploads a local file to the given remote path relative to the path of the run",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			runPath := args[0]
			remoteRelPath := args[1]

			// assert that RUN_PATH points to an actual rundb entry
			_, err := rundb.NewRunInfoFromPath(runPath)
			fail(err)

			remotePath := fmt.Sprintf("%s/%s", runPath, remoteRelPath)

			fail(wait(remotePath))
		},
	}

	return &cmd
}

func getArtifactCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "get-artifact RUN_PATH REMOTE_REL_PATH LOCAL_PATH",
		Short: "downloads an artifact(s) from a given path, relative to the run path",
		Args:  cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			runPath := args[0]
			remoteRelPath := args[1]
			localPath := args[2]

			// assert that RUN_PATH points to an actual rundb entry
			_, err := rundb.NewRunInfoFromPath(runPath)
			fail(err)

			remotePath := fmt.Sprintf("%s/%s", runPath, remoteRelPath)

			fail(download(localPath, remotePath))
		},
	}

	return &cmd
}

func setFinishedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-finished RUN_PATH",
		Short: "sets the run's status as finished",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runPath := args[0]

			ri, err := rundb.NewRunInfoFromPath(runPath)
			fail(err)

			ri.SetStatus(rundb.StatusFinished)
			fail(ri.Save())
		},
	}
}

func setErrorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-error RUN_PATH ERROR",
		Short: "sets the run's status as having an error",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			runPath := args[0]
			errStr := args[1]

			ri, err := rundb.NewRunInfoFromPath(runPath)
			fail(err)

			ri.SetStatus(rundb.StatusError)
			ri.Error = errStr
			fail(ri.Save())
		},
	}
}

func main() {
	rootCmd := &cobra.Command{Use: "rundb"}
	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(addArtifactCmd())
	rootCmd.AddCommand(getArtifactCmd())
	rootCmd.AddCommand(waitArtifactCmd())
	rootCmd.AddCommand(setFinishedCmd())
	rootCmd.AddCommand(setErrorCmd())
	// TODO: subcommand to add a result

	fail(rootCmd.Execute())
}

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
