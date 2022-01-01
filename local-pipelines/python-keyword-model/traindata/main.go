package main

import (
	"log"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use: "[sub]",
	}

	root.AddCommand(collectCmd)
	root.AddCommand(frequencyCmd)

	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}
