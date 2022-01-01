package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/scanner"
	"go/token"
	"io"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/codeexample"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

// This binary generates training data from the curated code examples
// and trains a new word2vec model with the new data.
func main() {
	var curatedSnippets, output string
	flag.StringVar(&curatedSnippets, "curated", "", "curated snippets emr file")
	flag.StringVar(&output, "output", "", "output file that contains the training data")
	flag.Parse()

	if flag.NFlag() != 2 {
		log.Fatalln("must specify -curated and -output")
	}

	file, err := os.Open(curatedSnippets)
	if err != nil {
		log.Fatal(err)
	}
	r := awsutil.NewEMRReader(file)

	w, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()

	for {
		_, value, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var cs codeexample.CuratedSnippet
		err = json.Unmarshal(value, &cs)
		if err != nil {
			log.Fatal(err)
		}

		_, err = fmt.Fprintln(w, normalize(cs.Curated.Snippet.Title))
		if err != nil {
			log.Fatalln("can't write title:", err)
		}

		prelude := tokenizeCode(cs.Curated.Snippet.Prelude)
		_, err = fmt.Fprintln(w, strings.Join(prelude, " "))
		if err != nil {
			log.Fatalln("can't write prelude:", err)
		}

		code := tokenizeCode(cs.Curated.Snippet.Code)
		_, err = fmt.Fprintln(w, strings.Join(code, " "))
		if err != nil {
			log.Fatalln("can't write code:", err)
		}
	}
}

func normalize(title string) string {
	title = strings.Trim(title, " \n")
	title = strings.ToLower(title)
	title = strings.Replace(title, "[", "", -1)
	title = strings.Replace(title, "]", "", -1)
	title = strings.Replace(title, "`s", "", -1)
	title = strings.Replace(title, "`", "", -1)
	return title
}

func tokenizeCode(code string) []string {
	buf := []byte(code)
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(buf))
	var lexer scanner.Scanner
	lexer.Init(file, buf, nil, 0)
	var words []string
	for {
		_, t, lit := lexer.Scan()
		if t == token.EOF {
			break
		}
		words = append(words, tokenToWords(t, lit)...)
	}
	return words
}

func tokenToWords(t token.Token, lit string) []string {
	switch t {
	case token.ILLEGAL, token.EOF:
		// Do not generate any words for these tokens
		return nil
	case token.COMMENT, token.STRING:
		return []string{lit}
	case token.IDENT:
		return []string{lit}
	default:
		return []string{t.String()}
	}
}
