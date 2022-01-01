package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/kiteco/kiteco/kite-server/metadata/service"
)

type server struct {
	deploymentID string
}

func (s *server) DeploymentID(context.Context, *empty.Empty) (*wrappers.StringValue, error) {
	return wrapperspb.String(s.deploymentID), nil
}

func main() {
	var tokenFile string
	flag.StringVar(&tokenFile, "tokenFile", "/run/secrets/kite-server-deployment-token", "File to load deployment token.")
	flag.Parse()

	f, err := os.Open(tokenFile)
	if err != nil {
		log.Fatalln("Error reading Kite Server deployment token", err)
	}
	defer f.Close()

	t, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalln("Error reading Kite Server deployment token", err)
	}

	h := sha256.New()
	h.Write(t)
	id := base64.StdEncoding.EncodeToString(h.Sum(nil))

	s := &server{deploymentID: id}
	gS := grpc.NewServer()
	service.RegisterMetadataServiceService(gS, &service.MetadataServiceService{
		DeploymentID: s.DeploymentID,
	})

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln("Error listening on port 8080", err)
	}
	gS.Serve(lis)
}
