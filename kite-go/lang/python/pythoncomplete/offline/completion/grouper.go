package completion

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/legacy"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

func groupNRCompletions(completions []data.NRCompletion, groupMap legacy.SignatureMap) {
	argSpec, argCount := getArgSpecInNRCompletion(completions)
	if argSpec == nil {
		return
	}

	completionMap(completions, func(completion *data.RCompletion) {
		desc, _ := legacy.GetSignatureDescription(completion.Snippet, argSpec, argCount)
		completion.Debug = completion.Debug.(legacy.MixCompletion).WithSignatureDescription(desc)
		groupMap[desc.Prototype] = append(groupMap[desc.Prototype], desc)
	})
}

func getArgSpecInNRCompletion(completions []data.NRCompletion) (*pythonimports.ArgSpec, int) {
	for _, comp := range completions {
		spec, argCount := legacy.GetArgSpecInComp(comp.RCompletion.Debug.(pythonproviders.MetaCompletion))
		if spec != nil {
			return spec, argCount
		}
	}
	return nil, -1
}
