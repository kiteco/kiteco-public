package kiteserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-server/metadata/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const timeout = 10 * time.Second

// GetHealth checks if the url is valid and reachable, and returns its deployment id and ping time in ms.
// an error is returned for empty or invalid urls.
func GetHealth(url string) (string, int64, error) {
	if url == "" {
		return "", 0, errors.New("empty authority")
	}

	parsedURL, err := ParseKiteServerURL(url)
	if err != nil {
		return "", 0, err
	}

	// use TLS for https:// urls, disable TLS for all other
	var opts []grpc.DialOption
	if parsedURL.Scheme == "https" {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	start := time.Now()
	conn, err := grpc.Dial(parsedURL.Host, opts...)
	if err != nil {
		log.Println(err)
		return "", 0, err
	}
	defer conn.Close()

	client := service.NewMetadataServiceClient(conn)
	id, err := doRequest(client)
	if err != nil {
		log.Println(err)
		return "", 0, err
	}
	return id, time.Since(start).Milliseconds(), nil
}

func doRequest(client service.MetadataServiceClient) (string, error) {
	var res string
	errc := kitectx.Go(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		var err error
		id, err := client.DeploymentID(ctx, &empty.Empty{})
		res = id.GetValue()
		return err
	})

	select {
	case err := <-errc:
		if err != nil {
			return "", err
		}
		return res, nil
	}
}

// ParseKiteServerURL takes a string, which represents a KTS endpoint
func ParseKiteServerURL(s string) (*url.URL, error) {
	var urlString string
	if strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") {
		urlString = s
	} else {
		urlString = "//" + s
	}

	// first, try to parse as-is to prefer existing schemes
	u, err := url.Parse(urlString)
	if err != nil || u.Scheme == "" {
		// no scheme: use http:// if a port is defined
		u, err = url.Parse("http://" + s) // Attach scheme since Parse is invalid without one
		if err != nil {
			return nil, errors.Errorf("unable to parse url %s: %v", s, err)
		}

		// use port 443 and https:// if no port is defined in the url
		if u.Port() == "" {
			u.Scheme = "https"
		}
	}

	if u.Hostname() == "" {
		return nil, errors.Errorf("Hostname cannot be empty")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, errors.Errorf("unsupported scheme %s", u.Scheme)
	}

	if u.Port() == "" {
		if port := defaultSchemePort(u.Scheme); port != "" {
			u.Host = fmt.Sprintf("%s:%s", u.Hostname(), port)
		}
	}

	return u, nil
}

func defaultSchemePort(name string) string {
	if name == "https" {
		return "443"
	}
	if name == "http" {
		return "80"
	}
	return ""
}
