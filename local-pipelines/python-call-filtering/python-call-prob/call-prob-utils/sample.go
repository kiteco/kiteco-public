package callprobutils

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"

	"github.com/kiteco/kiteco/kite-golib/pipeline"

	"github.com/kiteco/kiteco/local-pipelines/python-call-filtering/internal/utils"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/callprob"
)

// GetTrainSample computes features and label from a sampleInput to generate a complete TrainSample
func GetTrainSample(in SampleInputs, res utils.Resources, partialCall bool) (callprob.TrainSample, error) {
	modelIn := callprob.Inputs{
		RM:        res.RM,
		Cursor:    in.Cursor,
		RAST:      in.RAST,
		CallComps: in.CallComps,
		Sym:       in.Sym,
		ScopeSize: in.ScopeSize,
	}

	features, err := callprob.NewFeatures(modelIn)
	if err != nil {
		return callprob.TrainSample{}, pipeline.WrapErrorAsError("can't get features", err)
	}

	var compFeatures []callprob.CompFeatures
	var callComps []pythongraph.PredictedCall
	for i, call := range in.CallComps {
		if features.Comp[i].Skip {
			continue
		}
		compFeatures = append(compFeatures, features.Comp[i])
		callComps = append(callComps, call)

		// just the arguments, no parens
		var parts []string
		for _, arg := range call.Args {
			if arg.Stop {
				break
			}

			if arg.Name == "" {
				parts = append(parts, arg.Value)
			} else {
				parts = append(parts, fmt.Sprintf("%s=%s", arg.Name, arg.Value))
			}
		}
	}

	if len(compFeatures) == 0 {
		return callprob.TrainSample{}, pipeline.NewErrorAsError("no valid calls")
	}
	features.Comp = compFeatures

	labels, err := utils.GetLabelsOnStruct(in.UserCall, callComps, !partialCall, partialCall)
	if err != nil {
		return callprob.TrainSample{}, pipeline.WrapErrorAsError("unable to get label", err)
	}

	var compIDs []string
	for _, comp := range callComps {
		compIDs = append(compIDs, comp.String())
	}

	return callprob.TrainSample{
		Features: features,
		Labels:   labels,
		Meta: callprob.TrainSampleMeta{
			Hash:            in.Hash,
			Cursor:          in.Cursor,
			CompIdentifiers: compIDs,
		},
	}, nil
}

// InspectableSample contains all information about a training sample to help debugging the pipeline in the sample inspector
type InspectableSample struct {
	Truncated bool
	Sample    callprob.TrainSample
	Source    string
	UserCall  *pythonast.CallExpr
	UserTyped string
}

// SimulatePipeline takes a string as input and generate inspectable training sample from it
// It is used by the sample-inspector
func SimulatePipeline(codeBlock string, res utils.Resources) ([]InspectableSample, error) {
	rng := NewRNG(1)
	ins, err := GetCallsToPredict("", []byte(codeBlock), res, rng.Random())
	if err != nil {
		fmt.Println("Error while getting calls from the file : ", err)
	}
	var result []InspectableSample

	for _, c := range ins {
		preds, sym, scopeSize, err := res.PredictCall(c.Src2, c.Words2, c.RAST2, c.Call2)
		if err != nil {
			fmt.Println(err)
			continue
		}

		truncatedPreds := utils.TruncateCalls(preds)
		fullPreds := utils.KeepOnlyCompleteCalls(preds)

		truncatedSIN := SampleInputs{
			Hash:      c.Hash,
			Cursor:    int64(c.Call.LeftParen.End),
			RAST:      c.RAST2,
			Sym:       sym,
			UserTyped: c.Src[c.Call.LeftParen.End:c.Call.RightParen.Begin],
			UserCall:  c.Call,
			CallComps: truncatedPreds,
			ScopeSize: scopeSize,
		}
		truncatedSamples, err := GetTrainSample(truncatedSIN, res, true)
		if err != nil {
			return nil, err
		}

		fullSIN := SampleInputs{
			Hash:      c.Hash,
			Cursor:    int64(c.Call.LeftParen.End),
			RAST:      c.RAST2,
			Sym:       sym,
			UserTyped: c.Src[c.Call.LeftParen.End:c.Call.RightParen.Begin],
			UserCall:  c.Call,
			CallComps: fullPreds,
			ScopeSize: scopeSize,
		}
		fullSamples, err := GetTrainSample(fullSIN, res, false)
		if err != nil {
			return nil, err
		}

		result = append(result, InspectableSample{
			Truncated: true,
			Sample:    truncatedSamples,
			Source:    string(c.Src2),
			UserCall:  c.Call,
			UserTyped: string(truncatedSIN.UserTyped),
		})

		result = append(result, InspectableSample{
			Truncated: false,
			Sample:    fullSamples,
			Source:    string(c.Src2),
			UserCall:  fullSIN.UserCall,
			UserTyped: string(fullSIN.UserTyped),
		})

	}
	return result, nil
}
