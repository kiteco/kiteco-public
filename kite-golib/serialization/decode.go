package serialization

import (
	"compress/bzip2"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// Decoder is an interface that matches gob.Decoder, json.Decoder, and xml.Decoder
type Decoder interface {
	// Decode extracts an object from the stream
	Decode(interface{}) error
}

// ErrStop is a special value returned from handlers to cease processing
var ErrStop = errors.New("stop processing requested")

// decodeWith with extracts objects from the given decoder and passes them to the handler
func decodeWith(d Decoder, elemType reflect.Type, handler func(interface{}) error) error {
	for {
		elem := reflect.New(elemType).Interface()
		err := d.Decode(elem)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		err = handler(elem)
		if err == ErrStop {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// Decode loads a series of objects from a file. If the path ends with .gz or .bz2 then
// the contents will be decompressed. The encoding is then determined by the remaining file
// extension, which can be .json, .gob, .xml, or .emr.
//
//   var apples []Apple
//   err := serialization.Decode("/tmp/numbers.json.gz", func(apple *Apple) {
//     numbers = append(numbers, num.X)
//   })
func Decode(path string, handler interface{}) error {
	r, err := fileutil.NewCachedReader(path)
	if err != nil {
		return fmt.Errorf("error loading %s: %v", path, err)
	}
	defer r.Close()
	return decodeAs(r, path, handler)
}

// decodeAs is like Decode but uses the provided path to determine the compression and
// encoding used in the file.
func decodeAs(r io.Reader, path string, handler interface{}) error {
	inpath := path
	// Switch on compression
	switch {
	case strings.HasSuffix(path, ".gz"):
		path = strings.TrimSuffix(path, ".gz")
		rd, err := gzip.NewReader(r)
		if err != nil {
			return fmt.Errorf("error loading %s: %v", inpath, err)
		}
		defer rd.Close()
		r = rd
	case strings.HasSuffix(path, ".bz2"):
		path = strings.TrimSuffix(path, ".bz2")
		r = bzip2.NewReader(r)
	}

	// Switch on encoding
	var d Decoder
	switch {
	case strings.HasSuffix(path, ".json"):
		d = json.NewDecoder(r)
	case strings.HasSuffix(path, ".gob"):
		d = gob.NewDecoder(r)
	case strings.HasSuffix(path, ".xml"):
		d = xml.NewDecoder(r)
	case strings.HasSuffix(path, ".emr"):
		d = awsutil.NewEMRDecoder(r)
	default:
		return fmt.Errorf("could not find decoder for %s", inpath)
	}

	// Examine the function signature
	f := reflect.ValueOf(handler)

	if f.Kind() == reflect.Ptr {
		return d.Decode(handler)
	}
	if f.Kind() != reflect.Func {
		panic("expected a function or a pointer as last parameter")
	}

	funcType := f.Type()
	if funcType.NumIn() != 1 {
		panic("expected a function with one input parameter")
	}
	if funcType.NumOut() > 1 {
		panic("expected a function with zero or one output parameter")
	}
	ptrType := funcType.In(0)
	if ptrType.Kind() != reflect.Ptr {
		panic("expected function parameter to be a pointer")
	}
	elemType := ptrType.Elem()

	// Do the actual decoding
	err := decodeWith(d, elemType, func(x interface{}) error {
		ret := f.Call([]reflect.Value{reflect.ValueOf(x)})
		if len(ret) == 0 || ret[0].IsNil() {
			return nil
		}
		return ret[0].Interface().(error)
	})
	if err != nil {
		return fmt.Errorf("error decoding %s: %v", inpath, err)
	}
	return nil
}
