package legacy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/driver"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-golib/complete/data"

	"github.com/kiteco/kiteco/kite-golib/errors"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

// MixCompletion encapsulates all the information for the given completion at the time of mixing
type MixCompletion struct {
	driver.Completion
	sd *SignatureDescription
}

// WithSignatureDescription sets the SignatureDescription
func (mc MixCompletion) WithSignatureDescription(sd SignatureDescription) MixCompletion {
	mc.sd = &sd
	return mc
}

// MetaCompletion returns the MetaCompletion
func (mc MixCompletion) MetaCompletion() pythonproviders.MetaCompletion {
	return mc.Meta
}

// SignatureDescription describe a completion (what argument are filled, which of them are positional and placeholders
type SignatureDescription struct {
	Prototype       string
	Names           []string
	Positionals     []bool
	Placeholders    []bool
	NumConcreteArgs int
	DropCompletion  bool
}

// SignatureMap = map[string][]SignatureDescription
type SignatureMap = map[string][]SignatureDescription

func positionalArgCount(sd *SignatureDescription) uint {
	var result uint
	for _, p := range sd.Positionals {
		if p {
			result++
		}
	}
	return result
}

// GetArgSpecInComp looks in different metadata field to find the ArgSpec of the call being completed and the number
// or args already filled
func GetArgSpecInComp(comp pythonproviders.MetaCompletion) (*pythonimports.ArgSpec, int) {
	if comp.GGNNMeta != nil {
		return comp.GGNNMeta.ArgSpec, comp.GGNNMeta.NumOrigArgs
	}
	if comp.CallModelMeta != nil {
		return comp.CallModelMeta.ArgSpec, comp.CallModelMeta.NumOrigArgs
	}
	if comp.CallPatternMeta != nil {
		return comp.CallPatternMeta.ArgSpec, comp.CallPatternMeta.ArgumentCount
	}
	if comp.ArgSpecMeta != nil {
		return comp.ArgSpecMeta.ArgSpec, comp.ArgSpecMeta.ArgumentCount
	}
	return nil, -1
}

func isPlaceholder(s string, start int, placeholders []data.Selection) bool {
	end := start + len(s)
	for _, s := range placeholders {
		if start <= s.Begin && s.End <= end {
			return true
		}
	}
	return false
}

// GetSignatureDescription builds a signature description for a call completion
func GetSignatureDescription(completion data.Snippet, spec *pythonimports.ArgSpec, argOffset int) (SignatureDescription, error) {
	txt := completion.Text
	var start int
	// Reduce recall 4 and 5 on call model (but that was here to fix a bug when the completion contains the function name)
	//openParIdx := strings.Index(txt, "(")
	//if openParIdx != -1 {
	//	txt = txt[openParIdx+1:]
	//	start = openParIdx
	//}
	//closeParIdx := strings.Index(txt, ")")
	//if closeParIdx != -1 {
	//	txt = txt[:closeParIdx]
	//}
	splitComp := strings.Split(txt, ",")
	type argData struct {
		positional bool
		value      string
		name       string
	}
	var parts []argData

	onlyPositional := true
	var numConcreteArgs int
	for position, arg := range splitComp {
		arg = strings.TrimSpace(arg)
		if strings.Index(arg, data.PlaceholderBeginMark) == -1 {
			numConcreteArgs++
		}
		i := strings.Index(arg, "=")
		if i > -1 {
			//keyword arg
			onlyPositional = false
			name := arg[:i]
			var value string
			if isPlaceholder(arg[i+1:], start+i+1, completion.Placeholders()) {
				value = "[]"
			} else {
				value = arg[i+1:]
			}
			parts = append(parts, argData{false, value, name})

		} else {
			if !onlyPositional {
				return SignatureDescription{}, errors.New("Invalid signature, positional arg after a keyword arg")
			}
			name, err := getArgName(spec, uint(position+argOffset))
			var value string
			if isPlaceholder(arg, start, completion.Placeholders()) {
				value = "[]"
			} else {
				value = arg
			}
			if err != nil {
				return SignatureDescription{}, err
			}
			parts = append(parts, argData{true, value, name})

		}
		start += len(arg) + 1 //Move the start to the end of the current arg + 1 for the comma

	}
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].name < parts[j].name
	})
	var result SignatureDescription
	var prototype string
	for _, ad := range parts {
		result.Positionals = append(result.Positionals, ad.positional)
		result.Placeholders = append(result.Placeholders, ad.value == "[]")
		result.Names = append(result.Names, ad.name)
		prototype += ad.name
		if ad.value == "[]" {
			prototype += "_[]"
		} else {
			prototype += "_"
		}
		prototype += " "
	}
	result.Prototype = prototype
	result.NumConcreteArgs = numConcreteArgs

	return result, nil
}

func getArgName(spec *pythonimports.ArgSpec, i uint) (string, error) {
	args := spec.NonReceiverArgs()
	if int(i) >= len(args) {
		if spec.Vararg != "" {
			return fmt.Sprintf("%s_%d", spec.Vararg, i), nil
		}
		return "", errors.New("Illegal arg index (%d), the function prototype has only %d positional args", i, len(spec.Args))
	}
	return args[i].Name, nil
}

// CompletionNotInSigs checks that the completion is not a duplicate of a completion in the given signatures map
func CompletionNotInSigs(mc MixCompletion, groups SignatureMap) bool {
	if mc.sd == nil || groups == nil {
		return true
	}
	group := groups[mc.sd.Prototype]
	if len(group) <= 1 {
		return true
	}

	posArgCount := positionalArgCount(mc.sd)
	for _, oc := range group {
		if posArgCount < positionalArgCount(&oc) {
			return false
		}
	}
	return true
}
