package mixing

import (
	"encoding/json"
)

// Normalizer normalizes scores from different providers
type Normalizer struct {
	multipliers map[int]providerNormalizers
}

type providerNormalizers struct {
	normalizer   float64
	experimental float64
}

// NewNormalizer ...
func NewNormalizer() (Normalizer, error) {
	contents, err := Asset("serve/normalizers.json")
	if err != nil {
		return Normalizer{}, err
	}
	var multipliers map[int]providerNormalizers
	err = json.Unmarshal(contents, &multipliers)
	if err != nil {
		return Normalizer{}, err
	}
	normalizer := Normalizer{
		multipliers: multipliers,
	}
	return normalizer, nil
}

// Normalize ...
func (n Normalizer) Normalize(providerName int, score float64, experimental bool) float64 {
	normalizers, ok := n.multipliers[providerName]
	if !ok {
		return score
	}
	if experimental {
		return score * normalizers.experimental
	}
	return score * normalizers.normalizer
}
