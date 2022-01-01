package diskmap

import (
	"encoding/json"
	"errors"
)

var (
	// ErrNotSupported is returned when a Codec is given input that is not supported
	ErrNotSupported = errors.New("not supported")
)

// builder is an internal interface used by builders to create diskmaps. Its only used internaly
// to allow different builders to be used with Codecs as long as they satisfy this interface
type builder interface {
	Add(key string, value []byte) error
}

// Codec allows for abitrary serialization methods. It includes helper methods
// Add and Get to work with *Builder and *Map objects.
type Codec struct {
	Marshal   func(v interface{}) ([]byte, error)
	Unmarshal func(data []byte, v interface{}) error
}

// Add the provided key/obj pair to the Builder, marshalling with c.Marshal
func (c Codec) Add(b builder, key string, obj interface{}) error {
	buf, err := c.Marshal(obj)
	if err != nil {
		return err
	}
	return b.Add(key, buf)
}

// Get the obj associated with key in Map, obj unmarshalled using c.Unmarshal
func (c Codec) Get(m Getter, key string, obj interface{}) error {
	buf, err := m.Get(key)
	if err != nil {
		return err
	}
	err = c.Unmarshal(buf, obj)
	if err != nil {
		return err
	}

	return nil
}

// --

// JSON defines a JSON Codec
var JSON = Codec{
	Marshal: func(v interface{}) ([]byte, error) {
		return json.Marshal(v)
	},
	Unmarshal: func(data []byte, v interface{}) error {
		return json.Unmarshal(data, v)
	},
}

// --

// Raw defines a Codec that allows strings and []byte slices through directly to the Map/Builder.
var Raw = Codec{
	Marshal: func(v interface{}) ([]byte, error) {
		switch t := v.(type) {
		case []byte:
			return t, nil
		case string:
			return []byte(t), nil
		}
		return nil, ErrNotSupported
	},
	Unmarshal: func(buf []byte, v interface{}) error {
		switch data := v.(type) {
		case *string:
			*data = string(buf)
			return nil
		case *[]byte:
			*data = buf
			return nil
		}
		return ErrNotSupported
	},
}
