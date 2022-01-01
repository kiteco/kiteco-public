package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"strings"
	"time"

	sitter "github.com/kiteco/go-tree-sitter"
	gositter "github.com/kiteco/go-tree-sitter/golang"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/golang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer/treesitter"
	"github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/utils"
)

func parse(src []byte) *sitter.Tree {
	l := gositter.GetLanguage()
	parser := sitter.NewParser()
	parser.SetLanguage(l)
	return parser.Parse(src)
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

// Find one random site where prettified result does not match the original
func main() {
	localFiles, err := utils.LocalFiles(lexicalv0.NewLangGroup(lang.Golang), "")
	if err != nil {
		log.Fatalln("error retrieving local files")
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(localFiles), func(i, j int) {
		localFiles[i], localFiles[j] = localFiles[j], localFiles[i]
	})
iterate:
	for _, f := range localFiles {
		buf, err := ioutil.ReadFile(f)
		if err != nil {
			continue
		}

		// parse the original source
		oriTree := parse(buf)

		if isInvalidTree(oriTree) {
			log.Fatalln("invalid tree")
		}

		// prettify the source
		var target bytes.Buffer
		if _, err := golang.Prettify(
			&target,
			golang.DefaultPrettifyConfig,
			buf, 0, len(buf),
			oriTree.RootNode(),
		); err != nil {
			log.Fatal(err)
		}

		got := target.String()

		// Our prettify function doesn't print extra spaces and extra new lines
		// Remove them from source files to make things easier
		want := string(buf)
		for strings.Index(want, "\n\n") != -1 {
			want = strings.ReplaceAll(want, "\n\n", "\n")
		}
		for strings.Index(want, "  ") != -1 {
			want = strings.ReplaceAll(want, "  ", " ")
		}

		want = strings.TrimSpace(want)
		got = strings.TrimSpace(got)

		l := len(got)
		if len(want) < l {
			l = len(want)
		}
		for i := 0; i < l; i++ {
			if got[i] != want[i] {
				start := i - 100
				if start < 0 {
					start = 0
				}
				end := i + 10
				if end > l {
					end = l
				}
				fmt.Println("Found difference in", f)
				fmt.Println("============================ WANT ==========================")
				fmt.Println(want[start:end])
				fmt.Println("============================ GOT ===========================")
				fmt.Println(got[start:end])
				break iterate
			}
		}
	}
}
