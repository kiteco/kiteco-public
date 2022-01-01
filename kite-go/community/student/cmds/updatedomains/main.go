package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/community/student"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

const pathSwotTxt = "kite-go/community/student/cmds/updatedomains/swot.txt"

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	result := initDomainsMaps()
	fmt.Printf("List size : WhiteList %d blackList %d\n", len(result.WhiteList), len(result.BlackList))

	data, err := json.Marshal(result)
	fail(err)

	path := fmt.Sprintf("s3://kite-data/swot-student-domains/domain-lists_%s.json.gz", time.Now().Format("2006-01-02T15:04:05"))
	writer, err := awsutil.NewBufferedS3Writer(path)
	gWriter := gzip.NewWriter(writer)
	_, err = gWriter.Write(data)
	fail(err)
	fail(gWriter.Close())
	fail(writer.Close())
	fmt.Println("Domains list successfully uploaded on s3 to the path ", path)
}

func initDomainsMaps() student.DomainLists {
	filePath := pathSwotTxt
	if len(os.Args) == 2 {
		filePath = os.Args[1]
	}

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	BlackList := make(map[string]struct{})
	WhiteList := make(map[string]struct{})

	r := bufio.NewReader(file)
	for {
		line, _, err := r.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		l := string(line)
		if strings.HasPrefix(l, "-") {
			BlackList[l[1:]] = struct{}{}
		} else {
			WhiteList[l] = struct{}{}
		}
	}
	return student.DomainLists{
		WhiteList: WhiteList,
		BlackList: BlackList,
	}
}
