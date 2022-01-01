package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
)

func fail(message string, args ...interface{}) {
	fmt.Printf(message+"\n", args...)
	os.Exit(-1)
}

func main() {
	args := os.Args
	if len(args) < 2 {
		exe, _ := os.Executable()
		if len(args) == 1 {
			exe = args[0]
		}

		fail("usage: %s [-beta] isInstalled | detect | detectRunning | install | update | uninstall | isRunning | openFile\n", exe)
	}

	betaChannel := false
	for i, arg := range args {
		if arg == "-beta" {
			println("Using beta plugin channel...")
			args = append(args[0:i], args[i+1:]...)
			betaChannel = true
			break
		}
	}

	allPlugins, err := createPlugins(betaChannel)
	if err != nil {
		fail("error initializing plugins: %s\n", err.Error())
	}

	ctx := context.Background()

	switch args[1] {
	case "isInstalled":
		isInstalled(ctx, findPlugins(allPlugins))
	case "detect":
		detect(ctx, findPlugins(allPlugins))
	case "detectRunning":
		detectRunning(ctx, findPlugins(allPlugins))
	case "install":
		install(ctx, findPlugins(allPlugins))
	case "update":
		update(ctx, findPlugins(allPlugins))
	case "uninstall":
		uninstall(ctx, findPlugins(allPlugins))
	case "isRunning":
		isRunning(ctx, findPlugins(allPlugins))
	case "openFile":
		openFile(ctx, allPlugins, args[2:])
	default:
		fmt.Printf("unknown option: %s\n", args[1])
		os.Exit(-1)
	}
}

func openFile(ctx context.Context, allPlugins []editor.Plugin, args []string) {
	if len(args) < 2 || len(args) > 4 {
		fail("usage: editorID filePath [line] [editorPath]")
	}

	id := args[0]
	filePath := args[1]

	var line int
	var lineErr error
	if len(args) >= 3 {
		line, lineErr = strconv.Atoi(args[2])
	}

	var editorPath string
	if len(args) == 3 && lineErr != nil {
		editorPath = args[2]
	} else if len(args) == 4 {
		editorPath = args[3]
	}

	var plugin editor.Plugin
	for _, p := range allPlugins {
		if p.ID() == id {
			plugin = p
			break
		}
		for _, mgr := range allPlugins {
			if ids, ok := mgr.(internal.AdditionalIPluginDs); ok {
				if shared.StringsContain(ids.AdditionalIDs(), id) {
					plugin = mgr
					break
				}
			}
		}
	}
	if plugin == nil {
		fail("plugin manager not found for %s", id)
	}

	errChan, err := plugin.OpenFile(ctx, id, editorPath, filePath, line)
	if err != nil {
		fail("Failed to open file: %v", err)
	}

	// wait for the (background) process to finish
	if errChan != nil {
		<-errChan
	}
}

func findPlugins(allPlugins []editor.Plugin) []editor.Plugin {
	// all args after the command are treated as plugin IDs and are passed to the command handlers
	var usedPlugins []editor.Plugin
	if len(os.Args) > 2 {
		usedPlugins = []editor.Plugin{}
		for _, id := range os.Args[2:] {
			found := false
			for _, p := range allPlugins {
				if p.ID() == id {
					usedPlugins = append(usedPlugins, p)
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("unknown plugin id %s\n", id)
				os.Exit(-1)
			}
		}
	}

	// sort plugins by ID for stable output
	sort.Slice(usedPlugins, func(i, j int) bool {
		return strings.Compare(usedPlugins[i].ID(), usedPlugins[j].ID()) < 0
	})
	return usedPlugins
}

func isInstalled(ctx context.Context, plugins []editor.Plugin) {
	fmt.Println("Detecting installed state of editors...")
	for _, p := range plugins {
		if paths, err := p.DetectEditors(ctx); err != nil {
			fmt.Printf("\terror detecting %s: %s\n", p.ID(), err.Error())
		} else {
			for _, e := range shared.MapEditors(ctx, paths, p) {
				start := time.Now()
				isInstalled := p.IsInstalled(ctx, e.Path)
				duration := time.Since(start)

				fmt.Printf("\t%s\t%s\tisInstalled: %v\t(duration: %s)\n", p.ID(), e.Path, isInstalled, duration.String())
			}
		}
		fmt.Println()
	}
}

func detect(ctx context.Context, plugins []editor.Plugin) {
	fmt.Println("Detecting installed editors...")
	for _, p := range plugins {
		fmt.Printf("\tDetecting %s...\n", p.ID())
		start := time.Now()
		paths, err := p.DetectEditors(ctx)

		if err != nil {
			fmt.Printf("\terror detecting %s: %s\n", p.ID(), err.Error())
		} else {
			for _, e := range shared.MapEditors(ctx, paths, p) {
				fmt.Printf("\t%s\t%s\t%s\t%s\n", p.ID(), e.Path, e.Version, e.Compatibility)
			}
		}

		duration := time.Since(start)
		fmt.Printf("Time needed for detection: %s\n", duration.String())
		fmt.Println()
	}
}

func detectRunning(ctx context.Context, plugins []editor.Plugin) {
	fmt.Println("Detecting running editors...")
	for _, p := range plugins {
		fmt.Printf("\tDetecting %s...\n", p.ID())
		start := time.Now()
		paths, err := p.DetectRunningEditors(ctx)

		if err != nil {
			fmt.Printf("\terror detecting %s: %s\n", p.ID(), err.Error())
		} else {
			for _, e := range shared.MapEditors(ctx, paths, p) {
				fmt.Printf("\t%s\t%s\t%s\t%s\n", p.ID(), e.Path, e.Version, e.Compatibility)
			}
		}

		duration := time.Since(start)
		fmt.Printf("Time needed for detection: %s\n", duration.String())
		fmt.Println()
	}
}

func install(ctx context.Context, plugins []editor.Plugin) {
	fmt.Println("Installing plugins...")
	for _, p := range plugins {
		if cfg := p.InstallConfig(ctx); cfg.Running && !cfg.InstallWhileRunning {
			fmt.Println("\tskipping install because editor is still running")
			continue
		}

		fmt.Printf("\tInstalling %s...\n", p.ID())
		if paths, err := p.DetectEditors(ctx); err != nil {
			fmt.Printf("\terror detecting %s: %s\n", p.ID(), err.Error())
		} else {
			for _, e := range shared.MapEditors(ctx, paths, p) {
				start := time.Now()
				err = p.Install(ctx, e.Path)
				duration := time.Since(start)

				if err != nil {
					fmt.Printf("\terror installing %s: %s\n", p.ID(), err.Error())
				} else {
					fmt.Printf("\tsuccessfully installed %s for editor %s (duration: %s)\n", p.ID(), e.Path, duration.String())
				}
			}
		}
		fmt.Println()
	}
}

func update(ctx context.Context, plugins []editor.Plugin) {
	fmt.Println("Updating plugins...")
	for _, p := range plugins {
		if cfg := p.InstallConfig(ctx); cfg.Running && !cfg.UpdateWhileRunning {
			fmt.Println("\tskipping update because editor is still running")
			continue
		}

		fmt.Printf("\tUpdating %s...\n", p.ID())
		if paths, err := p.DetectEditors(ctx); err != nil {
			fmt.Printf("\terror detecting %s: %s\n", p.ID(), err.Error())
		} else {
			for _, e := range shared.MapEditors(ctx, paths, p) {
				start := time.Now()
				err = p.Update(ctx, e.Path)
				duration := time.Since(start)

				if err != nil {
					fmt.Printf("\terror updating %s: %s\n", p.ID(), err.Error())
				} else {
					fmt.Printf("\tsuccessfully updated %s for editor %s (duration: %s)\n", p.ID(), e.Path, duration.String())
				}
			}
		}
		fmt.Println()
	}
}

func uninstall(ctx context.Context, plugins []editor.Plugin) {
	fmt.Println("Uninstalling plugins...")
	for _, p := range plugins {
		if cfg := p.InstallConfig(ctx); cfg.Running && !cfg.UninstallWhileRunning {
			fmt.Printf("\tskipping uninstall because editor is still running")
			continue
		}

		fmt.Printf("\tUninstalling %s...\n", p.ID())
		if paths, err := p.DetectEditors(ctx); err != nil {
			fmt.Printf("\terror detecting %s: %s\n", p.ID(), err.Error())
		} else {
			for _, e := range shared.MapEditors(ctx, paths, p) {
				start := time.Now()
				err = p.Uninstall(ctx, e.Path)
				duration := time.Since(start)

				if err != nil {
					fmt.Printf("\terror uninstalling %s: %s\n", p.ID(), err.Error())
				} else {
					fmt.Printf("\tsuccessfully uninstalled %s for editor %s (duration: %s)\n", p.ID(), e.Path, duration.String())
				}
			}
		}
		fmt.Println()
	}
}

func isRunning(ctx context.Context, plugins []editor.Plugin) {
	w := tabwriter.NewWriter(os.Stdout, 4, 0, 1, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "Checking isRunning state of plugins...")
	for _, p := range plugins {
		cfg := p.InstallConfig(ctx)
		fmt.Fprintf(w, "\t%s\t%v\n", p.ID(), cfg.Running)
	}
}
