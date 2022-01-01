package decisiontree

import (
	"encoding/json"
	"io"
)

// Load is a function
func Load(r io.Reader) (*Ensemble, error) {
	var ensemble Ensemble
	rd := json.NewDecoder(r)
	err := rd.Decode(&ensemble)
	if err != nil {
		return nil, err
	}
	return &ensemble, nil
}
