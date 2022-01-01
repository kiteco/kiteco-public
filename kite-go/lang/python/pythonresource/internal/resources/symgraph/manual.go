package symgraph

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/stringutil"
	"github.com/tinylib/msgp/msgp"
)

// String is a structural analog of string
type String string

// EncodeMsg implements msgp.Encodable
func (s String) EncodeMsg(en *msgp.Writer) error {
	return en.WriteString(string(s))
}

// DecodeMsg implements msgp.Decodable
func (s *String) DecodeMsg(dc *msgp.Reader) error {
	b, err := dc.ReadStringAsBytes(nil)
	if err != nil {
		return err
	}
	(*s) = String(stringutil.InternBytes(b))
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (s String) Msgsize() int {
	return msgp.StringPrefixSize + len(string(s))
}

// ChildMap is a structural analog of Node.Children
type ChildMap map[uint64]NodeRef

// EncodeMsg implements msgp.Encodable
func (m ChildMap) EncodeMsg(en *msgp.Writer) error {
	if err := en.WriteMapHeader(uint32(len(m))); err != nil {
		return err
	}
	for k, v := range m {
		if err := en.WriteString(stringutil.FromUint64(k)); err != nil {
			return err
		}
		if err := v.EncodeMsg(en); err != nil {
			return err
		}
	}
	return nil
}

// DecodeMsg implements msgp.Decodable
func (m *ChildMap) DecodeMsg(dc *msgp.Reader) error {
	sz, err := dc.ReadMapHeader()
	if err != nil {
		return err
	}

	if (*m) == nil {
		(*m) = make(ChildMap, sz)
	} else if len((*m)) > 0 {
		for k := range *m {
			delete((*m), k)
		}
	}

	for sz > 0 {
		sz--

		b, err := dc.ReadStringAsBytes(nil)
		if err != nil && err != msgp.ErrShortBytes {
			return err
		}

		var v NodeRef
		err = v.DecodeMsg(dc)
		if err != nil {
			return err
		}

		if len(b) == 0 {
			// this happens in opencv-contrib-python TODO(naman)
			continue
		}
		(*m)[stringutil.ToUint64Bytes(b)] = v
	}
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (m ChildMap) Msgsize() int {
	sz := msgp.MapHeaderSize
	if m != nil {
		for k, v := range m {
			sz += msgp.StringPrefixSize + len(stringutil.FromUint64(k)) + v.Msgsize()
		}
	}
	return sz
}
