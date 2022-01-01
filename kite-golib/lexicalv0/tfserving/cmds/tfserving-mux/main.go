package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	_ "github.com/kiteco/kiteco/kite-golib/status"

	serving_proto "github.com/kiteco/kiteco/kite-golib/protobuf/tensorflow/serving"
	"google.golang.org/grpc"
)

func main() {
	var (
		grpcPort    int
		forwardTo   string
		contextSize int
		logAll      bool
		logErrors   bool
	)
	flag.IntVar(&grpcPort, "grpc-port", 8600, "port to listen for gRPC requests")
	flag.StringVar(&forwardTo, "forward-to", "", "tfserving server to forward requests to")
	flag.IntVar(&contextSize, "context-size", 300, "resize incoming contexts to this size (0 for noop)")
	flag.BoolVar(&logAll, "log-all", false, "log every request")
	flag.BoolVar(&logErrors, "log-errors", false, "log only requests that error")
	flag.Parse()

	// Handle semantic overlap
	if logAll {
		logErrors = false
	}

	go http.ListenAndServe(":8601", nil)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("creating grpc server")
	grpcServer := grpc.NewServer()

	log.Println("registering server")
	server, err := newServer(forwardTo, logAll, logErrors, contextSize)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	serving_proto.RegisterPredictionServiceServer(grpcServer, server)

	log.Printf("listening on port %d...", grpcPort)
	go grpcServer.Serve(lis)

	log.Println("ready!")
	select {}
}
