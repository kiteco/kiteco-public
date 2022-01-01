package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-golib/serialization"
	"github.com/kr/pretty"
)

func fail(msg interface{}, parts ...interface{}) {
	fmt.Printf(fmt.Sprintf("%v", msg)+"\n", parts...)
	os.Exit(1)
}

func main() {
	var args struct {
		Input  string `arg:"positional,required"`
		Output string `arg:"positional,required"`
		Limit  int
		Print  bool
	}
	arg.MustParse(&args)

	// open output
	enc, err := serialization.NewEncoder(args.Output)
	if err != nil {
		fail(err)
	}
	defer enc.Close()

	// open input
	f, err := os.Open(args.Input)
	if err != nil {
		fail(err)
	}

	// setup scanner
	scanner := bufio.NewScanner(f)
	scanner.Buffer(nil, 2<<24) // 16MB

	// drop the first two lines
	scanner.Scan()
	scanner.Scan()

	// decode posts one by one
	var count, ignored int
	for scanner.Scan() && args.Limit == 0 || count < args.Limit {
		var post stackoverflow.XMLPost
		err := xml.Unmarshal(scanner.Bytes(), &post)
		if err != nil {
			ignored++
			fmt.Println(err, "ignoring...")
			continue
		}

		if !strings.Contains(post.Tags, "python") {
			continue
		}

		err = enc.Encode(post)
		if err != nil {
			fail(err)
		}

		count++
		if count%1000 == 0 {
			fmt.Println(count)
		}

		if args.Print {
			pretty.Println(post)
		}
	}
	if err := scanner.Err(); err != nil {
		fail(err)
	}
	if ignored > 0 {
		fmt.Printf("ignored %d lines\n", ignored)
	}

	fmt.Printf("wrote %d posts to %s\n", count, args.Output)
}
