package predict

import (
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
)

// LoadModelAssets ...
func LoadModelAssets(modelPath string, group lexicalv0.LangGroup) (*lexicalv0.FileEncoder, HParams, SearchConfig, error) {
	vocabPath := fileutil.Join(modelPath, "ident-vocab-entries.bpe")
	encoder, err := lexicalv0.NewFileEncoder(vocabPath, group)
	if err != nil {
		return nil, HParams{}, SearchConfig{}, err
	}

	paramsPath := fileutil.Join(modelPath, "config.json")
	params, err := NewHParams(paramsPath)
	if err != nil {
		return nil, HParams{}, SearchConfig{}, err
	}

	searchPath := fileutil.Join(modelPath, "searchconfig.json")
	search, err := NewSearchConfig(searchPath)
	if err != nil {
		return nil, HParams{}, SearchConfig{}, err
	}

	return encoder, params, search, nil
}

func selectSearchConfig(config SearchConfig, in Inputs) SearchConfig {
	if (in.SearchConfig == SearchConfig{}) {
		return config
	}
	return in.SearchConfig
}
