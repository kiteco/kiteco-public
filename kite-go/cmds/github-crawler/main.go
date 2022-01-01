package main

import (
	"flag"
	"log"
	"runtime"
)

func main() {
	var (
		// args for master node
		masterMode bool
		port       string
		input      string
		fetched    string

		// args for crawler node
		n         int
		hostPort  string
		outputdir string
	)

	flag.BoolVar(&masterMode, "master", false, "run as master node")
	flag.StringVar(&port, "port", ":2020", "port master listens on")
	flag.StringVar(&input, "input", "", "csv file with repos to crawl")
	flag.StringVar(&fetched, "fetched", "fetched.json", "local state tracking whats already been fetched")

	flag.IntVar(&n, "n", runtime.NumCPU(), "concurrency")
	flag.StringVar(&hostPort, "hostPort", "", "host-port of master node")
	flag.StringVar(&outputdir, "outputdir", "", "dir to stage the crawl before it gets uploaded")

	flag.Parse()

	if masterMode {
		if input == "" {
			log.Fatalln("./github-crawler -input=<repos csv> [-fetched=<progress snapshot> -port=<listen port>]")
		}

		master(input, fetched, port)
		return
	}

	if hostPort == "" {
		log.Fatalln("./github-crawler -hostPort=<host:port> [-n=<concurrency>]")
	}

	crawl(n, hostPort, outputdir)
}
