package main

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/mtacconf"
	"github.com/kiteco/kiteco/local-pipelines/python-mtac-filtering/internal/utils"
)

func getTrainSample(in sampleInputs, res utils.Resources) (mtacconf.TrainSample, error) {
	modelIn := getModelInputs(in, res)

	features, err := mtacconf.NewFeatures(modelIn)
	if err != nil {
		return mtacconf.TrainSample{}, fmt.Errorf("can't get features: %v", err)
	}

	label, err := utils.GetLabel(in.UserTyped, in.Idents)
	if err != nil {
		return mtacconf.TrainSample{}, fmt.Errorf("unable to fetch label %v", err)
	}

	return mtacconf.TrainSample{
		Features: features,
		Label:    label,
		Meta: mtacconf.TrainSampleMeta{
			Hash:            in.Hash,
			Cursor:          in.Cursor,
			CompIdentifiers: in.Idents,
		},
	}, nil
}

func getModelInputs(in sampleInputs, res utils.Resources) mtacconf.Inputs {
	return mtacconf.Inputs{
		RM:     res.RM,
		Cursor: in.Cursor,
		Words:  in.Words,
		RAST:   in.RAST,
		Comps:  in.Completions,
	}
}
