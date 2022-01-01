package jsonrpc2

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// Derived from:
// https://github.com/golang/tools/blob/master/internal/jsonrpc2/stream.go

// TransportConnection represents a generic transport-layer connection for reading/writing jsonrpc2 messages
type TransportConnection interface {
	Read() ([]byte, error)
	Write([]byte) error
}

// ReaderWriterConnection is a TransportConnection that uses file stdio as the transport mechanism
type ReaderWriterConnection struct {
	in       *bufio.Reader
	outMutex sync.Mutex
	out      io.Writer
}

// NewReaderWriterConnection ...
func NewReaderWriterConnection(in io.Reader, out io.Writer) TransportConnection {
	return &ReaderWriterConnection{
		in:  bufio.NewReader(in),
		out: out,
	}
}

// Read attempts to read a jsonrpc2 packet (with headers) from the given connection
func (conn *ReaderWriterConnection) Read() ([]byte, error) {
	var length int64
	for {
		line, err := conn.in.ReadString('\n')
		if err != nil {
			return nil, errors.New("unable to read header line %q", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			// we're done with headers
			break
		}
		colonIndex := strings.IndexRune(line, ':')
		if colonIndex < 0 {
			return nil, errors.New("invalid header line %q", line)
		}
		header, value := line[:colonIndex], strings.TrimSpace(line[colonIndex+1:])
		switch header {
		case "Content-Length": //TODO what is proper behavior when there are multiple Content-Length headers? We use the last one given
			length, err = strconv.ParseInt(value, 10, 32)
			if err != nil || length <= 0 {
				return nil, errors.New("invalid Content-Length: %v", value)
			}
		}
	}
	if length == 0 {
		return nil, errors.New("missing Content-Length header")
	}
	data := make([]byte, length)
	_, err := io.ReadFull(conn.in, data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Write attempts to write the given slice of bites out through the given connection
func (conn *ReaderWriterConnection) Write(data []byte) error {
	conn.outMutex.Lock()
	defer conn.outMutex.Unlock()
	_, err := fmt.Fprintf(conn.out, "Content-Length: %v\r\n\r\n", len(data))
	if err == nil {
		_, err = conn.out.Write(data)
	}
	return err
}
