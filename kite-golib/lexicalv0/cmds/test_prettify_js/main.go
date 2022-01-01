package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	sitter "github.com/kiteco/go-tree-sitter"
	jssitter "github.com/kiteco/go-tree-sitter/javascript"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/javascript"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer/treesitter"
)

var defaultConf = &javascript.Config{
	ArrayBracketSpacing:     true,
	CommaSpacingAfter:       true,
	Indent:                  2,
	KeySpacingAfterColon:    true,
	KeywordSpacingBefore:    true,
	KeywordSpacingAfter:     true,
	ObjectCurlyNewline:      1,
	ObjectPropertyNewline:   1,
	Semicolon:               true,
	SpaceBeforeBlocks:       true,
	SpaceInfixOps:           true,
	SpaceUnaryOpsWords:      true,
	StatementNewline:        true,
	SwitchColonSpacingAfter: true,
}

var (
	flagSemi              = flag.Bool("semi", false, "Insert semicolons")
	flagOutDir            = flag.String("dir", "", "Output directory for original and prettified output files (default: temp dir)")
	flagIgnoreSemi        = flag.Bool("ignore-semi", false, "Ignore semicolons in diff")
	flagIgnoreEmpty       = flag.Bool("ignore-empty", false, "Ignore empty statements in diff")
	flagIgnoreInvalidFile = flag.Bool("ignore-invalid-file", false, "Ignore files that can't be parsed successfully in original form")
	flagCmpLevel          = flag.Int("level", 0, "Diff level: 0=node type only, 1=terminal content (ignore jsx_text whitespace), 2=full content (default: 0)")
)

func main() {
	flag.Parse()

	defaultConf.Semicolon = *flagSemi
	outDir := os.TempDir()
	if *flagOutDir != "" {
		outDir = *flagOutDir
	}

	var r io.Reader

	switch flag.NArg() {
	case 1:
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		r = f

	case 0:
		r = os.Stdin

	default:
		log.Fatal("USAGE: test_prettify_js [FILE]")
	}

	src, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	// parse the original source
	oriTree := parse(src)
	var oriBuf bytes.Buffer
	if err := renderTree(&oriBuf, src, oriTree); err != nil {
		log.Fatal(err)
	}

	if *flagIgnoreInvalidFile && isInvalidTree(oriTree) {
		return
	}

	// prettify the source
	var buf bytes.Buffer
	if _, err := javascript.Prettify(&buf, *defaultConf, src, 0, len(src), oriTree.RootNode()); err != nil {

		log.Fatal(err)
	}

	// parse the prettified output
	prettyTree := parse(buf.Bytes())
	var prettyBuf bytes.Buffer
	if err := renderTree(&prettyBuf, buf.Bytes(), prettyTree); err != nil {
		log.Fatal(err)
	}

	// compare both parse results
	oriTreePath, prettyTreePath := filepath.Join(outDir, "original.tree.js"), filepath.Join(outDir, "pretty.tree.js")
	if err := ioutil.WriteFile(oriTreePath, oriBuf.Bytes(), 0600); err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(prettyTreePath, prettyBuf.Bytes(), 0600); err != nil {
		log.Fatal(err)
	}
	prettySrcPath := filepath.Join(outDir, "prettyfied.source.js")
	if err := ioutil.WriteFile(prettySrcPath, buf.Bytes(), 0600); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("For clearer diffing:\n> diff -bB %s %s\n", oriTreePath, prettyTreePath)

	if want, got := oriBuf.String(), prettyBuf.String(); want != got {
		fmt.Printf("\nparse trees differ, see rendered source:\n> cat %s\n", prettySrcPath)
		os.Exit(1)
	}
}

func isInvalidTree(tree *sitter.Tree) bool {
	var invalid bool
	treesitter.Inspect(tree.RootNode(), func(n *sitter.Node) bool {
		if n == nil || invalid {
			return false
		}
		if typ := n.Type(); typ == "ERROR" || typ == "MISSING" {
			invalid = true
		}
		return true
	})
	return invalid
}

func parse(src []byte) *sitter.Tree {
	lang := jssitter.GetLanguage()
	parser := sitter.NewParser()
	parser.SetLanguage(lang)
	return parser.Parse(src)
}

func renderTree(w io.Writer, src []byte, tree *sitter.Tree) error {
	root := tree.RootNode()
	d := &debugger{
		w:   w,
		src: src,
	}
	treesitter.Walk(d, root)
	return d.err
}

type debugger struct {
	w     io.Writer
	err   error
	src   []byte
	depth int
}

func (d *debugger) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil || d.err != nil {
		d.depth--
		return nil
	}

	d.depth++
	d.err = printTreeNode(d.w, d.depth, d.src, n)
	return d
}

var rxRemoveEmpty = regexp.MustCompile(`(\s*\(empty_statement\))+(\s*)`)

func printTreeNode(w io.Writer, depth int, src []byte, n *sitter.Node) error {
	const (
		maxContent = 300
		indent     = ".  "
	)

	if *flagIgnoreSemi && n.Type() == ";" && n.Content(src) == ";" {
		return nil
	}
	if *flagIgnoreEmpty && n.Type() == "empty_statement" {
		return nil
	}

	content := n.Content(src)
	if n.Type() == "jsx_text" && *flagCmpLevel == 1 {
		// ignore whitespace in jsx_text
		content = strings.TrimSpace(content)
	}
	if strings.TrimSpace(content) == "" {
		content = strconv.Quote(content)
	} else {
		content = strings.Join(strings.Fields(content), " ")
	}
	if len(content) > maxContent {
		content = content[:maxContent-3] + "..."
	}

	switch *flagCmpLevel {
	case 0:
		content = ""
	case 1:
		// content only for terminals/string (regex) literals
		if typ := n.Type(); typ != "string" && typ != "template_string" && typ != "regex" && n.ChildCount() > 0 {
			content = ""
		}
	}

	nodeStr := n.String()
	if *flagIgnoreEmpty && strings.Contains(nodeStr, "(empty_statement)") {
		nodeStr = rxRemoveEmpty.ReplaceAllString(nodeStr, "$2")
	}
	if len(nodeStr) > maxContent {
		nodeStr = nodeStr[:maxContent-3] + "..."
	}
	prefix := strings.Repeat(indent, depth-1)
	_, err := fmt.Fprintf(w, "%s%d: %q | %s | %s\n",
		prefix, n.Symbol(), n.Type(), nodeStr, content)
	return err
}
