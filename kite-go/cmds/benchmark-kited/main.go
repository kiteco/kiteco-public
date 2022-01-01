package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/proto"
)

var (
	logPrefix = fmt.Sprintf("[%s] ", "benchmark-kited")
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
}

func main() {
	var (
		codeDir    string
		iterations int
	)

	flag.StringVar(&codeDir, "codeDir", "", "directory containing code to use to simulate events")
	flag.IntVar(&iterations, "iterations", 1, "number of iterations per user")
	flag.Parse()

	if codeDir == "" {
		flag.Usage()
		os.Exit(0)
	}

	var files []string
	err := filepath.Walk(codeDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalln("could not read code dir:", err)
	}

	fmt.Printf("==== found %d code files\n", len(files))
	fmt.Printf("==== starting event stream in 5 seconds... ")
	time.Sleep(time.Second)
	fmt.Printf("4...")
	time.Sleep(time.Second)
	fmt.Printf("3...")
	time.Sleep(time.Second)
	fmt.Printf("2...")
	time.Sleep(time.Second)
	fmt.Printf("1...")
	time.Sleep(time.Second)
	fmt.Printf("\n")

	benchmark(iterations, files)
}

func benchmark(iterations int, files []string) {
	conn := connect()
	defer conn.Close()

	for iter := 0; iter < iterations; iter++ {
		// Select a file at random
		filename := files[rand.Intn(len(files))]
		buf, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Printf("could not read file %s: %s\n", filename, err)
			continue
		}
		fmt.Println("generating events from", filename)

		for off := 0; off < len(buf); off++ {
			// Simulate approximately 80WPM typing speed (upper bound)
			time.Sleep(150 * time.Millisecond)

			// Build the event
			ev := eventAtOffset(iter, off, buf)
			if off == 0 {
				// First event should be a focus event
				ev.Action = proto.String("focus")
			}

			bufEv, err := json.Marshal(ev)
			if err != nil {
				log.Fatalln("could not marshal event:", err)
			}

			_, err = conn.Write(bufEv)
			if err != nil {
				log.Fatalln("error writing:", err)
			}
			if off%100 == 0 {
				fmt.Printf(":")
			} else if off%10 == 0 {
				fmt.Printf(".")
			}
		}
		fmt.Printf("\n")
	}
}

func eventAtOffset(iter, off int, buf []byte) *event.Event {
	return &event.Event{
		Filename:  proto.String(fmt.Sprintf("tempfile%d.py", iter)),
		Source:    proto.String("sublime-text"),
		Action:    proto.String("edit"),
		Text:      proto.String(string(buf[:off])),
		Timestamp: proto.Int64(time.Now().UnixNano()),
		Selections: []*event.Selection{
			&event.Selection{
				Start: proto.Int64(int64(off)),
				End:   proto.Int64(int64(off)),
			},
		},
	}
}

const (
	bufferSize = 4 << 20
)

func connect() *net.UnixConn {
	addr := &net.UnixAddr{
		Name: os.ExpandEnv("$HOME/.kite/kite.sock"),
		Net:  "unixgram",
	}

	conn, err := net.DialUnix("unixgram", nil, addr)
	if err != nil {
		log.Fatalln("could not get connection to kite.sock:", err)
	}
	err = conn.SetWriteBuffer(bufferSize)
	if err != nil {
		log.Fatalln("cound not set write buffer:", err)
	}

	return conn
}
