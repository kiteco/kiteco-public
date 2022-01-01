package popularsignatures

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/tinylib/msgp/msgp"
)

// Entities indexes signature patterns by symbol
type Entities map[pythonimports.Hash]Entity

// EncodeMsg implements msgp.Encodable
func (r Entities) EncodeMsg(en *msgp.Writer) error {
	if err := en.WriteMapHeader(uint32(len(r))); err != nil {
		return err
	}
	for k, v := range r {
		if err := en.WriteUint64(uint64(k)); err != nil {
			return err
		}
		if err := v.EncodeMsg(en); err != nil {
			return err
		}
	}
	return nil
}

// DecodeMsg implements msgp.Decodable
func (r *Entities) DecodeMsg(dc *msgp.Reader) error {
	sz, err := dc.ReadMapHeader()
	if err != nil {
		return err
	}

	if (*r) == nil {
		(*r) = make(Entities, sz)
	} else if len((*r)) > 0 {
		for k := range *r {
			delete((*r), k)
		}
	}

	for sz > 0 {
		sz--
		u, err := dc.ReadUint64()
		if err != nil {
			return err
		}
		var v Entity
		err = v.DecodeMsg(dc)
		if err != nil {
			return err
		}
		(*r)[pythonimports.Hash(u)] = v
	}
	return nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (r Entities) Msgsize() int {
	sz := msgp.MapHeaderSize
	if r != nil {
		for _, v := range r {
			sz += msgp.StringPrefixSize + msgp.Uint64Size + v.Msgsize()
		}
	}
	return sz
}
