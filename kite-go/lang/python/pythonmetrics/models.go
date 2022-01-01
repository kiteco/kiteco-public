package pythonmetrics

import (
	"errors"
	"go/token"
	"math/rand"

	"github.com/kiteco/kiteco/kite-golib/stringindex"
)

// ReferenceResolutionComparison compare the resolution level of a reference between IntelliJ and Kite
type ReferenceResolutionComparison struct {
	intelliJResolutionLevel ResolutionLevel
	kiteResolutionLevel     ResolutionLevel
}

func (comp ReferenceResolutionComparison) String() string {
	delta := comp.kiteResolutionLevel - comp.intelliJResolutionLevel
	if delta > 0 {
		return "%%% Kite wins : " + comp.kiteResolutionLevel.String() + " > " + comp.intelliJResolutionLevel.String()
	} else if delta < 0 {
		return "### IntelliJ wins : " + comp.intelliJResolutionLevel.String() + " > " + comp.kiteResolutionLevel.String()
	} else {
		return "••• Same resolution level : " + comp.kiteResolutionLevel.String()
	}
}

// ResolutionLevel is an enum of the different level of resolution for a reference
type ResolutionLevel int

const (
	// Unknown is a reference not resolved
	Unknown ResolutionLevel = iota
	// DuckType is a reference resolved by inference from usages
	DuckType
	// UnionType corresponds to a reference resolved to multiple types
	UnionType
	// Known if used when the type of the reference completely known
	Known
)

func newResolutionLevel(resLevel string) (ResolutionLevel, error) {
	switch resLevel {
	case "unknown":
		return Unknown, nil
	case "duckType":
		return DuckType, nil
	case "unionType":
		return UnionType, nil
	case "known":
		return Known, nil
	default:
		return 0, errors.New("ResolutionLevel unknown: " + resLevel)
	}
}

func (resLevel ResolutionLevel) String() string {
	switch resLevel {
	case Unknown:
		return "unknown type"
	case DuckType:
		return "duck type"
	case UnionType:
		return "union type"
	case Known:
		return "known type"
	default:
		return "ResolutionLevel unknown"
	}
}

// ReferenceType contains additional information about the type of a resolved reference in IntelliJ
type ReferenceType struct {
	ResolutionLevel string `json:"resolution_level"`
	Type            string `json:"type"`
	TypeOfType      string `json:"type_of_type"`
}

// ReferenceInfo describe a reference resolution in IntelliJ
type ReferenceInfo struct {
	Start         token.Pos     `json:"start"`
	End           token.Pos     `json:"end"`
	Resolved      bool          `json:"resolved"`
	Text          string        `json:"text"`
	ReferenceType ReferenceType `json:"type"`
	FoundInKite   bool
}

// ReferenceComparisonOnline contains all the details of a comparison of a reference resolution between Kite and IntelliJ
// When used in offline mode, some sensitive information are stored (user types, extract of source code)
type ReferenceComparisonOnline struct {
	IntelliJSymbolResolved  bool            `json:"intellij_symbol_resolved"`
	IntelliJResolutionLevel ResolutionLevel `json:"intellij_resolution_level"`
	IntelliJTypeOfType      string          `json:"intellij_type_of_type"`
	KiteSymbolResolved      bool            `json:"kite_symbol_resolved"`
	KiteResolutionLevel     ResolutionLevel `json:"kite_resolution_level"`
	KiteASTType             string          `json:"kite_ast_type"`
	KiteValueType           string          `json:"kite_value_type"`
}

// ReferenceComparison contains additional fields for the case where privacy is not an issue (offline metrics is done on open source project
// (these fields are only populated for the offline metrics for privacy reason)
type ReferenceComparison struct {
	OnlineFields ReferenceComparisonOnline `json:"online_fields"`
	Text         string                    `json:"text"`
	Begin        token.Pos                 `json:"begin"`
	End          token.Pos                 `json:"end"`
	KiteValue    string                    `json:"kite_value"`
	KiteSymbol   string                    `json:"kite_symbol"`
	IntelliJType string                    `json:"intellij_type"`
	Filename     string                    `json:"filename"`
}

// SampleTag implements pipeline.Sample
func (ReferenceComparison) SampleTag() {}

// ReferenceMap is an alias for the map used in the matching algorithm (to match a reference between IntelliJ and Kite)
type ReferenceMap map[string]*ReferenceInfo

// GetReferenceMap builds a Reference map from a list of reference and the content of the file
// The offset needs to be converted from runes (intelliJ offset) to bytes (Kite offset)
// The sampling is only done if 0 < samplingRate < 1
func GetReferenceMap(references []*ReferenceInfo, fileContent []byte, samplingRate float64, randomSeed int64) ReferenceMap {
	result := make(ReferenceMap, len(references))
	var rnd *rand.Rand
	if samplingRate <= 0 {
		samplingRate = 1
	} else if samplingRate < 1 {
		rnd = rand.New(rand.NewSource(randomSeed))
	}
	converter := stringindex.Converter{Bytes: fileContent}
	for _, ref := range references {
		if samplingRate >= 1 || rnd.Float64() < samplingRate {
			ref.Start = token.Pos(converter.BytesFromRunes(int(ref.Start)))
			ref.End = token.Pos(converter.BytesFromRunes(int(ref.End)))
			result[refKey(ref.Start, ref.End)] = ref
		}
	}
	return result
}
