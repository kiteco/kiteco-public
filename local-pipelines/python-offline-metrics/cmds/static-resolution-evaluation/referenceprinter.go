package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmetrics"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
)

type referencePrinter struct {
	depth        int
	indent       string
	positions    bool
	w            io.Writer
	referenceMap map[pythonast.Expr]pythontype.Value
	nodeMap      map[pythonast.Expr]pythonmetrics.ReferenceResolutionComparison
	lineMap      *linenumber.Map
}

func (p *referencePrinter) Visit(n pythonast.Node) pythonast.Visitor {
	if n == nil {
		p.depth--
	} else {
		prefix := strings.Repeat(p.indent, p.depth)
		if p.positions {
			if p.lineMap != nil {
				prefix = fmt.Sprintf("[%4d...%4d - line %3d]", n.Begin(), n.End(), p.lineMap.Line(int(n.Begin()))+1) + prefix
			} else {
				prefix = fmt.Sprintf("[%4d...%4d]", n.Begin(), n.End()) + prefix
			}
		}
		suffix := "    "
		if expr, ok := n.(pythonast.Expr); ok {
			refType := p.nodeMap[expr]
			suffix += refType.String()
		}
		_, err := fmt.Fprintln(p.w, prefix+pythonast.String(n)+suffix)
		maybeQuit(err)
		p.depth++
	}
	return p
}

// printReferences writes a textual representation of syntax tree to the given writer,
// including begin and end positions for each node and information on how each symbol is resolved in IntelliJ and Kite.
func printReferences(root pythonast.Node, w io.Writer, indent string, referenceMap map[pythonast.Expr]pythontype.Value, nodeMap map[pythonast.Expr]pythonmetrics.ReferenceResolutionComparison, sourceFile string) {
	var lm *linenumber.Map
	if sourceFile != "" {
		content, err := ioutil.ReadFile(sourceFile)
		maybeQuit(err)

		w.Write(content)
		w.Write([]byte(fmt.Sprintf("\n\n\nSource File : %v", sourceFile)))
		w.Write([]byte("Enhanced AST : \n\n"))
		lm = linenumber.NewMap(content)
	}

	printer := referencePrinter{
		w:            w,
		indent:       indent,
		positions:    true,
		referenceMap: referenceMap,
		nodeMap:      nodeMap,
		lineMap:      lm,
	}
	pythonast.Walk(&printer, root)
}

func printEnhancedRAST(sourceFolder string, rast *pythonanalyzer.ResolvedAST, filename string, astOutputFolder string, nodeMap map[pythonast.Expr]pythonmetrics.ReferenceResolutionComparison) {
	outputFile := strings.TrimPrefix(filename, sourceFolder)
	outputFile = fileutil.Join(astOutputFolder, outputFile+".ast.txt")
	if !awsutil.IsS3URI(outputFile) {
		maybeQuit(os.MkdirAll(filepath.Dir(outputFile), os.ModePerm))
	}
	file, err := fileutil.NewBufferedWriter(outputFile)
	maybeQuit(err)
	defer file.Close()
	printReferences(rast.Root, file, "  ", rast.References, nodeMap, filename)
}
