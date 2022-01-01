package rettypes

import (
	"encoding/json"
	"io"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/returntypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

// DistReturns encapsulates a distribution and a map from contained paths to their return type paths
// The return type paths may only be partially validated and should be validated by the consumer of this data.
type DistReturns struct {
	Dist    keytypes.Distribution `json:"dist"`
	Returns returntypes.Entities  `json:"returns"`
}

// EncodeAll encodes a sequence of DistReturns to w
func EncodeAll(w io.Writer, all []DistReturns) error {
	enc := json.NewEncoder(w)
	for _, dr := range all {
		if err := enc.Encode(dr); err != nil {
			return err
		}
	}
	return nil
}

// DecodeAll decodes a sequence of DistReturns from r
func DecodeAll(r io.Reader) ([]DistReturns, error) {
	var all []DistReturns
	dec := json.NewDecoder(r)
	for {
		var dr DistReturns
		err := dec.Decode(&dr)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		all = append(all, dr)
	}
	return all, nil
}
