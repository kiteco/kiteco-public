package main

import (
	"fmt"
	"go/token"
	"io/ioutil"
	"os"
	"strings"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

func findStmt(ast pythonast.Node, cursor token.Pos) pythonast.Stmt {
	var stmt pythonast.Stmt
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		if n == nil {
			return false
		}
		if n.Begin() > cursor || cursor > n.End() {
			return false
		}

		if n, ok := n.(pythonast.Stmt); ok {
			stmt = n
		}
		_, isbad := n.(*pythonast.BadStmt)
		return !isbad
	})
	return stmt
}

// compare positions
func cmp(a, b token.Position) int {
	switch {
	case b.Line < a.Line:
		return -1
	case b.Line > a.Line:
		return 1
	case b.Offset < a.Offset:
		return -1
	case b.Offset > a.Offset:
		return 1
	default:
		return 0
	}
}

func main() {
	var args struct {
		Src     string `arg:"positional"`
		Verbose bool   `arg:"-v"`
	}
	arg.MustParse(&args)

	// get the source
	src := args.Src
	if len(src) == 0 {
		buf, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		src = string(buf)
	}

	// find the cursor position
	cursor := strings.Index(src, "$")
	if cursor == -1 {
		fmt.Println("missing $ in input")
		os.Exit(1)
	}
	src = src[:cursor] + src[cursor+1:]
	srcbuf := []byte(src)

	// tokenize
	alltokens, err := pythonscanner.Scan(srcbuf)
	if err != nil {
		fmt.Println("error tokenizing:", err)
		os.Exit(1)
	}

	// parse
	cursorPos := token.Pos(cursor)
	ast, err := pythonparser.Parse(kitectx.Background(), srcbuf, pythonparser.Options{
		Approximate: true,
		Cursor:      &cursorPos,
	})
	if err != nil && ast == nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if args.Verbose {
		pythonast.PrintPositions(ast, os.Stdout, "  ")
	}

	// find the statement containing the cursor
	stmt := findStmt(ast, cursorPos)
	if stmt == nil {
		fmt.Println("no enclosing statement")
		os.Exit(1)
	}

	fmt.Println(string(src[stmt.Begin():stmt.End()]))

	// find tokens comprising the statement
	stmtbegin := stmt.Begin()
	var tokens []pythonscanner.Word
	for _, t := range alltokens {
		if stmtbegin <= t.Begin && t.Begin < cursorPos {
			tokens = append(tokens, t)
		}
	}

	var ts []pythonscanner.Token
	ts = append(ts, pythonscanner.LiteralTokens...)
	ts = append(ts, pythonscanner.OperatorTokens...)
	ts = append(ts, pythonscanner.KeywordTokens...)

	// try each possible token
	var allowed []pythonscanner.Token
	for _, t := range ts {
		pseudo := append(tokens, pythonscanner.Word{
			Begin: cursorPos,
			End:   cursorPos,
			Token: t,
		})

		// must add EOF
		pseudo = append(pseudo, pythonscanner.Word{
			Begin: cursorPos + 1,
			End:   cursorPos + 1,
			Token: pythonscanner.EOF,
		})

		_, err := pythonparser.ParseWords(kitectx.Background(), srcbuf, pseudo, pythonparser.Options{
			ErrorMode: pythonparser.FailFast,
		})
		errs, _ := err.(errors.Errors)

		accepted := true
		if errs != nil && errs.Slice()[0].(pythonscanner.PosError).Pos <= cursorPos {
			// error at or before cursor - this token is invalid here
			accepted = false
		}
		if accepted {
			allowed = append(allowed, t)
		}
	}

	for _, t := range allowed {
		fmt.Println("  ", t)
	}
}
