package pythonproviders

import (
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/lexicalproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

// RenderMeta encapsulates metadata for rendering symbol information for the completions
// TODO(juan): this is pretty nasty
type RenderMeta struct {
	Referent pythontype.Value `json:"-"`
}

// CallModelMeta encapsulates metadata for call completions
type CallModelMeta struct {
	FunctionSym     pythonresource.Symbol
	NumArgs         int
	NumOrigArgs     int
	CallProb        float64
	ArgSpec         *pythonimports.ArgSpec
	NumConcreteArgs int
	Call            *pythongraph.PredictedCall
}

// IdentModelMeta encapsulates metadata for attribute/name ("identifier") completions
type IdentModelMeta struct {
	MTACConfSkip bool // false if the confidence score exceeds the cutoff for the relevant scenario. Skip a completion if true.
}

// TraditionalMeta is metadata for traditional completions
type TraditionalMeta struct {
	Situation string
}

// CallPatternMeta is the metadata for completion from popular pattern (call pattern) Provider
type CallPatternMeta struct {
	ArgSpec       *pythonimports.ArgSpec
	ArgumentCount int
}

// ArgSpecMeta is the metadata for completion from ArgSpec Provider
type ArgSpecMeta struct {
	ArgSpec       *pythonimports.ArgSpec
	ArgumentCount int
}

// DictMeta is metadata for Dict Provider
type DictMeta struct {
	// AttributeToSubscript indicates the completion replaces a attribute lookup with a subscript
	AttributeToSubscript bool
}

// EmptyCallMeta contains info on completion generated by EmptyCallProvider
type EmptyCallMeta struct {
	IsTypeKind bool
}

// MixingMeta contains which provider the completion is from
type MixingMeta struct {
	DistanceFromRoot int
	Provider         ProviderJSON `json:"provider"`
	// Hide completion can be set to true to remove completion during the mixing
	HideCompletion bool `json:"hide_completion"`
	// DoNotCompose block collection to compose this completion with a parent
	// The completion will be collected only if it is applied directly to the user buffer state
	DoNotCompose bool `json:"do_not_compose"`
}

// GGNNMeta contains meta data needed for running GGNNNModel Provider
type GGNNMeta struct {
	Predictor                     *pythongraph.PredictorNew
	Call                          *pythongraph.PredictedCall
	ArgSpec                       *pythonimports.ArgSpec
	NumOrigArgs                   int
	Debug                         string
	SpeculationPlaceholderPresent bool
}

// LexicalFiltersMeta for semantic filtering of lexical completions
type LexicalFiltersMeta struct {
	InvalidArgument    bool
	InvalidAssignment  bool
	InvalidAttribute   bool
	HasBadStmt         bool
	InvalidClassDef    bool
	InvalidFunctionDef bool
	InvalidImport      bool
}

// MetaCompletion pairs a completion with metadata used for rendering and/or mixing
type MetaCompletion struct {
	data.Completion

	RenderMeta RenderMeta `json:"render_meta"`

	Provider data.ProviderName               `json:"provider"`
	Source   response.EditorCompletionSource `json:"source"`

	Score             float64 `json:"score"`
	ExperimentalScore float64 `json:"experimental_score"`
	NormalizedScore   float64 `json:"normalized_score"`

	ArgSpecMeta     *ArgSpecMeta     `json:"arg_spec_meta"`
	CallPatternMeta *CallPatternMeta `json:"call_pattern_meta"`
	CallModelMeta   *CallModelMeta   `json:"call_model_meta"`
	AttrModelMeta   *IdentModelMeta  `json:"attr_model_meta"`
	NameModelMeta   *IdentModelMeta  `json:"name_model_meta"`
	ExprModelMeta   *IdentModelMeta  `json:"expr_model_meta"`
	// KeywordModelMeta is true if the completion is from the keyword model
	KeywordModelMeta bool `json:"keyword_model_meta"`
	// TraditionalMeta is the value of the traditionally completed name/attribute/import
	TraditionalMeta *TraditionalMeta `json:"traditional_meta"`
	DictMeta        *DictMeta        `json:"dict_meta"`
	EmptyCallMeta   *EmptyCallMeta   `json:"empty_call_meta"`

	// MixingMeta is the meta info used in mixing.
	MixingMeta MixingMeta `json:"mixing_meta"`

	// GGNNMeta contains info related to GGNN model evaluation, and particularly the predictor that allows to call Expand method on a completion
	GGNNMeta *GGNNMeta `json:"ggnn_meta"`

	LexicalFiltersMeta *LexicalFiltersMeta
	LexicalMeta        *lexicalproviders.LexicalMeta
	LexicalMetrics     interface{}

	FromSmartProvider bool
	IsServer          bool
}
