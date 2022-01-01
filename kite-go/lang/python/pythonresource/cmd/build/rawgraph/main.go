package main

import (
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/rawgraph/types"
	"github.com/spf13/cobra"
)

func fail(err error) {
	if err != nil {
		log.Fatalln("[FATAL]", err)
	}
}

func validate(cmd *cobra.Command, args []string) {
	inp := args[0]
	out := args[1]

	var dat types.ExplorationData

	r, err := os.Open(inp)
	fail(err)
	defer r.Close()
	fail(dat.Decode(r))

	validateData(&dat)

	w, err := os.Create(out)
	fail(err)
	defer w.Close()
	fail(dat.Encode(w))
}

var validateCmd = cobra.Command{
	Use:   "graph INPUT VALID_OUTPUT",
	Short: "validate and fix the given pkgexploration data",
	Args:  cobra.ExactArgs(2),
	Run:   validate,
}

func main() {
	fail(validateCmd.Execute())
}
