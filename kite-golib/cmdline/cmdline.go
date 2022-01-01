package cmdline

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	arg "github.com/alexflint/go-arg"
)

// Command represents an action that can be run from the command line
type Command struct {
	Name     string
	Synopsis string
	Args     Handler
}

// Handler represents a function that gets called for an action
type Handler interface {
	Handle() error
}

// Validator is the interface for custom validation of command line arguments
type Validator interface {
	Validate() error
}

func prog() string {
	if len(os.Args) > 0 {
		return filepath.Base(os.Args[0])
	}
	return "program"
}

func writeUsage(w io.Writer, cmds ...Command) {
	fmt.Fprintf(w, "Usage: %s COMMAND [ARGS]\n", prog())
	fmt.Fprintf(w, "Command can be one of:\n")
	for _, cmd := range cmds {
		fmt.Fprintf(w, "  %-20s %s\n", cmd.Name, cmd.Synopsis)
	}
	fmt.Fprintf(w, "  %-20s %s\n", "help", "display this help and exit")
	fmt.Fprintf(w, "  %-20s %s\n", "help COMMAND", "display help for command and exit")
}

// MustDispatch dispatches one of the commands
func MustDispatch(cmds ...Command) *arg.Parser {
	if len(os.Args) < 2 {
		writeUsage(os.Stdout, cmds...)
		fmt.Println("\nError: no command provided")
		os.Exit(1)
	}

	var help bool
	action := os.Args[1]
	if action == "help" {
		if len(os.Args) < 3 {
			// write help for overall command
			writeUsage(os.Stdout, cmds...)
			fmt.Println("\nFor help on a specific command use ")
			os.Exit(0)
		}
		help = true
		action = os.Args[2]
	}

	// Get the action
	var cmd *Command
	for _, c := range cmds {
		if c.Name == action {
			cmd = &c
			break
		}
	}
	if cmd == nil {
		writeUsage(os.Stdout, cmds...)
		fmt.Println("\nError: unknown command", action)
		os.Exit(1)
	}

	// Create parser for this command
	config := arg.Config{
		Program: prog() + " " + action,
	}
	parser, err := arg.NewParser(config, cmd.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Show help for this command if requested
	if help {
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	// Parse the command line args
	err = parser.Parse(os.Args[2:])
	if err != nil {
		parser.Fail(err.Error())
	}

	// Validate
	if v, ok := cmd.Args.(Validator); ok {
		if err := v.Validate(); err != nil {
			parser.Fail(err.Error())
		}
	}

	// Dispatch the handler
	if err := cmd.Args.Handle(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return parser
}
