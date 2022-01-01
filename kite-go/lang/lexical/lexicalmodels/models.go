package lexicalmodels

import (
	"fmt"
	"log"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kiteserver"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

// Type specifies the type of model to use for a particular ModelConfig
type Type string

// Specifies the prediction modality of the model, currently
// the choices are tfpredictor (beam in go) and tfsearcher (beam in tf)
var (
	TFPredictorType Type = "tfpredictor"
	TFSearcherType  Type = "tfsearcher"
)

// ModelConfig ...
type ModelConfig struct {
	Type       Type
	Lang       lexicalv0.LangGroup
	ModelPath  string
	TFServing  predict.TFServingOptions
	RemoteOnly bool
}

// ModelOptions ...
type ModelOptions struct {
	TextMiscGroup ModelConfig
	TextWebGroup  ModelConfig
	TextJavaGroup ModelConfig
	TextCGroup    ModelConfig
}

// LocalModelOptions only use local models
var LocalModelOptions = ModelOptions{
	TextMiscGroup: ModelConfig{
		Type:      TFPredictorType,
		Lang:      lexicalv0.MiscLangsGroup,
		ModelPath: "s3://kite-data/run-db/2020-10-21T15:48:52Z_lexical-model-experiments/out_text__python-go-php-ruby-bash_lexical_context_512_embedding_180_layer_4_head_6_vocab_13500_steps_25000_batch_160",
	},
	TextWebGroup: ModelConfig{
		Type:      TFPredictorType,
		Lang:      lexicalv0.WebGroup,
		ModelPath: "s3://kite-data/run-db/2020-10-07T18:29:44Z_lexical-model-experiments/out_text__javascript-jsx-vue-css-html-less-typescript-tsx_lexical_context_512_embedding_180_layer_4_head_6_vocab_13500_steps_25000_batch_160",
	},
	TextJavaGroup: ModelConfig{
		Type:      TFPredictorType,
		Lang:      lexicalv0.JavaPlusPlusGroup,
		ModelPath: "s3://kite-data/run-db/2020-10-08T05:09:31Z_lexical-model-experiments/out_text__java-scala-kotlin_lexical_context_512_embedding_180_layer_4_head_6_vocab_13500_steps_25000_batch_160",
	},
	TextCGroup: ModelConfig{
		Type:      TFPredictorType,
		Lang:      lexicalv0.CStyleGroup,
		ModelPath: "s3://kite-data/run-db/2020-10-09T05:43:16Z_lexical-model-experiments/out_text__c-cpp-objectivec-csharp_lexical_context_512_embedding_180_layer_4_head_6_vocab_20000_steps_25000_batch_160",
	},
}

// DefaultRemoteHost is the default Kite Cloud Server host
var DefaultRemoteHost = "tfserving.kite.com:8085"

// WithRemoteModels sets remote model (TFServing) configuration
func (opts ModelOptions) WithRemoteModels(host string) ModelOptions {
	parsedHost, err := kiteserver.ParseKiteServerURL(host)

	var hostUsed, schemeUsed string
	if err == nil {
		hostUsed = parsedHost.Host
		schemeUsed = parsedHost.Scheme
	} else {
		hostUsed = host
		schemeUsed = "http"
	}

	allLangsOpts := predict.NewTFServingOptions(
		host,
		"all-langs-large",
		lexicalv0.AllLangsGroup,
		fmt.Sprintf("%s://%s/model-assets/text__python-go-javascript-jsx-vue-css-html-less-typescript-tsx-java-scala-kotlin-c-cpp-objectivec-csharp-php-ruby-bash/", schemeUsed, hostUsed),
	)
	opts.TextMiscGroup.TFServing = allLangsOpts
	opts.TextCGroup.TFServing = allLangsOpts
	opts.TextJavaGroup.TFServing = allLangsOpts
	opts.TextWebGroup.TFServing = allLangsOpts
	return opts
}

// RemoteOnly bypasses the tiered local/remote predictor, useful for testing/debugging
func (opts ModelOptions) RemoteOnly() ModelOptions {
	opts.TextMiscGroup.RemoteOnly = true
	opts.TextCGroup.RemoteOnly = true
	opts.TextJavaGroup.RemoteOnly = true
	opts.TextWebGroup.RemoteOnly = true
	return opts
}

// ClearRemoteModels removes remote model configuration
func (opts ModelOptions) ClearRemoteModels() ModelOptions {
	opts.TextMiscGroup.TFServing = predict.TFServingOptions{}
	opts.TextCGroup.TFServing = predict.TFServingOptions{}
	opts.TextJavaGroup.TFServing = predict.TFServingOptions{}
	opts.TextWebGroup.TFServing = predict.TFServingOptions{}
	return opts
}

// DefaultModelOptions ...
var DefaultModelOptions = LocalModelOptions

// GetDefaultModelOptions given a language
func GetDefaultModelOptions(language lexicalv0.LangGroup) (ModelConfig, error) {
	switch language.Lexer {
	case lang.Text:
		switch {
		case language.Equals(lexicalv0.MiscLangsGroup):
			return DefaultModelOptions.TextMiscGroup, nil
		case language.Equals(lexicalv0.WebGroup):
			return DefaultModelOptions.TextWebGroup, nil
		case language.Equals(lexicalv0.JavaPlusPlusGroup):
			return DefaultModelOptions.TextJavaGroup, nil
		case language.Equals(lexicalv0.CStyleGroup):
			return DefaultModelOptions.TextCGroup, nil
		default:
			return ModelConfig{}, errors.New("no default model path found")
		}
	default:
		return ModelConfig{}, errors.New("no default model path found")
	}
}

// Models ...
type Models struct {
	TextMiscGroup Model
	TextWebGroup  Model
	TextJavaGroup Model
	TextCGroup    Model
}

// NewModels ...
func NewModels(opts ModelOptions) (*Models, error) {
	// Use errors.Errors so we can accumulate errors. The main reason for this is so that
	// datadeps can capture all required data for the TFServingSearcher case. The models will
	// error out during datadeps generation if it cannot connect to the server -- however
	// all S3 resources will have been fetched before the connection is attempted/fails.
	var modelErr errors.Errors

	misc, err := newModelBase(opts.TextMiscGroup)
	if err != nil {
		modelErr = errors.Append(modelErr, errors.Wrapf(err, "text misc model"))
	}

	web, err := newModelBase(opts.TextWebGroup)
	if err != nil {
		modelErr = errors.Append(modelErr, errors.Wrapf(err, "text web model"))
	}

	java, err := newModelBase(opts.TextJavaGroup)
	if err != nil {
		modelErr = errors.Append(modelErr, errors.Wrapf(err, "text java model"))
	}

	c, err := newModelBase(opts.TextCGroup)
	if err != nil {
		modelErr = errors.Append(modelErr, errors.Wrapf(err, "text c model"))
	}

	return &Models{
		TextMiscGroup: withStatus(misc),
		TextWebGroup:  withStatus(web),
		TextJavaGroup: withStatus(java),
		TextCGroup:    withStatus(c),
	}, modelErr
}

// Return combined model for language
func newModelBase(opts ModelConfig) (ModelBase, error) {
	var err error

	// Configure Remote
	var remoteModel *predict.TFServingSearcher
	if !opts.TFServing.Empty() {
		remoteModel, err = predict.NewTFServingSearcher(opts.TFServing)
		if opts.RemoteOnly {
			return remoteModel, err
		}
	}

	// Configure Local
	var model ModelBase
	switch opts.Type {
	case TFPredictorType:
		model, err = predict.NewPredictor(opts.ModelPath, opts.Lang)
	case TFSearcherType:
		model, err = predict.NewTFSearcherFromS3(opts.ModelPath, opts.Lang)
	}
	if err != nil {
		return nil, err
	}

	model = predict.NewTFCombinedPredictor(model, remoteModel)
	return model, err
}

// UpdateRemote updates the remote models
func (m *Models) UpdateRemote(opts ModelOptions) {
	updateRemote(m.TextMiscGroup.Base(), opts.TextMiscGroup)
	updateRemote(m.TextCGroup.Base(), opts.TextCGroup)
	updateRemote(m.TextJavaGroup.Base(), opts.TextJavaGroup)
	updateRemote(m.TextWebGroup.Base(), opts.TextWebGroup)
}

func updateRemote(m ModelBase, opts ModelConfig) {
	var err error
	var model *predict.TFServingSearcher
	if !opts.TFServing.Empty() {
		model, err = predict.NewTFServingSearcher(opts.TFServing)
		if err != nil {
			log.Println("error updating remote:", err)
		}
	}
	pred, ok := m.(*predict.TFCombinedPredictor)
	if !ok {
		return
	}
	pred.Update(model)
}

// Reset unloads the model
func (m *Models) Reset() {
	m.TextMiscGroup.Unload()
	m.TextWebGroup.Unload()
	m.TextJavaGroup.Unload()
	m.TextCGroup.Unload()
}

func (m *Models) languageToModels(language lang.Language) []Model {
	switch language {
	case lang.Golang, lang.Python, lang.PHP, lang.Ruby, lang.Bash:
		return []Model{m.TextMiscGroup}
	case lang.JavaScript, lang.JSX, lang.Vue:
		return []Model{m.TextWebGroup}
	case lang.CSS, lang.HTML, lang.Less, lang.TypeScript, lang.TSX:
		return []Model{m.TextWebGroup}
	case lang.Java, lang.Scala, lang.Kotlin:
		return []Model{m.TextJavaGroup}
	case lang.C, lang.Cpp, lang.CSharp, lang.ObjectiveC:
		return []Model{m.TextCGroup}
	default:
		return nil
	}
}

// ResetLanguage unloads the model for a specific language
func (m *Models) ResetLanguage(language lang.Language) {
	for _, model := range m.languageToModels(language) {
		model.ResetLanguage(language)
	}
}

// IsLoaded returns whether the model associated with the extension is loaded
// TODO: Currently we have more than one model associated with an extension
//       So we return false if none of them is loaded
func (m *Models) IsLoaded(fext string) bool {
	for _, model := range m.languageToModels(lang.FromExtension(fext)) {
		if model.IsLoaded() {
			return true
		}
	}
	return false
}

// LanguageGroupDeprecated ...
// NOTE: this is deprecated, please do not use this
func LanguageGroupDeprecated(language lang.Language) lang.Language {
	switch language {
	case lang.Golang:
		return lang.Golang
	case lang.JavaScript, lang.JSX, lang.Vue:
		return lang.JavaScript
	case lang.Python:
		return lang.Python
	case lang.Text:
		return lang.Text
	default:
		return lang.Unknown
	}
}
