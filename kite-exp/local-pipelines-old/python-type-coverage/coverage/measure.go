package main

import (
	"fmt"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
)

var (
	measureCmd = cmdline.Command{
		Name:     "measure",
		Synopsis: "Measure coverage for static analysis",
		Args: &measureArgs{
			LibDepth: 1,
		},
	}
)

type measureArgs struct {
	Corpus   string `arg:"positional"`
	Trace    bool
	LibDepth int
}

func (args *measureArgs) Handle() error {
	start := time.Now()

	var tooLarge, parseErrs, files, added int64
	var numAttrs, numResolvedBases, numMembersFound int64
	wp := walkParams{
		Corpus:       args.Corpus,
		LibraryDepth: args.LibDepth,
	}
	if err := walk(wp, func(sources map[string]sourceFile, collector *collector, stats batchStats) error {
		tooLarge += stats.TooLarge
		parseErrs += stats.ParseErrors
		files += stats.Files
		added += stats.Added
		for _, file := range sources {
			pythonast.Inspect(file.AST, func(n pythonast.Node) bool {
				attr, isAttr := n.(*pythonast.AttributeExpr)
				if !isAttr {
					return true
				}
				numAttrs++

				base, found := collector.exprs[attr.Value]
				if !found || base == nil {
					return true
				}
				numResolvedBases++

				res, err := pythontype.AttrNoCtx(base, attr.Attribute.Literal)
				if err != nil || res.Value() == nil {
					return true
				}
				numMembersFound++
				return true
			})
		}
		if args.Trace {
			fmt.Printf("%s\n", stats.Trace)
		}
		fmt.Printf("Done with %s, took %v to process %d files, %d were too large, %d contained parse errors, %d files added to batch\n",
			stats.Corpus, stats.ProcessingTime, stats.Files, stats.TooLarge, stats.ParseErrors, stats.Added)
		return nil
	}); err != nil {
		return fmt.Errorf("error walking corpus `%s`: %v", args.Corpus, err)
	}

	fracBasesResolved := float64(numResolvedBases) / float64(numAttrs)
	fracMembers := float64(numMembersFound) / float64(numAttrs)

	fmt.Println("Done! Took", time.Since(start))
	fmt.Printf("Corpus contained %d files, %d too large, %d had parse errors, %d added\n",
		files, tooLarge, parseErrs, added)
	fmt.Printf("Found %d attribute expressions\n", numAttrs)
	fmt.Printf("  %.2f%% (%d) had bases that resolved\n", 100.*fracBasesResolved, numResolvedBases)
	fmt.Printf("  %.2f%% (%d) rhs were in lhs completions\n", 100.*fracMembers, numMembersFound)

	return nil
}
