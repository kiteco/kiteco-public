package localtraining

import (
	"log"
	"math/rand"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/bpe"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

// Params ...
type Params struct {
	TrainRatio       float64
	ValidateRatio    float64
	SplitType        SplitType
	NumGo            int
	VocabIters       int
	VocabInit        VocabInit
	WeightedSampling bool
	TrainSampleRate  float64
}

// Inputs ...
type Inputs struct {
	Language lexicalv0.LangGroup
	Seed     int64
	Files    []string

	GlobalModelPath string

	// TODO: unclear where this should go,
	// we could use a default for each language
	// but it is helpful to set this to control memory
	// consumption so leaving it here for now
	ContextSize int
}

// Trainer ...
type Trainer struct {
	params Params

	in                Inputs
	allFilePaths      []string
	trainFilePaths    []string
	validateFilePaths []string

	originalEncoder    *lexicalv0.FileEncoder
	originalEmbeddings [][]float32
}

// NewTrainer ...
func NewTrainer(params Params, in Inputs) (Trainer, error) {
	train, validate, _, err := Split(
		in.Files, in.Seed, params.SplitType, params.TrainRatio, params.ValidateRatio, 0,
	)

	if err != nil {
		return Trainer{}, errors.Wrapf(err, "unable to split files")
	}

	predictor, err := predict.NewTFPredictorFromS3(in.GlobalModelPath, in.Language)
	if err != nil {
		return Trainer{}, errors.Wrapf(err, "unable to load model")
	}
	defer predictor.Unload()

	// fetch initial token embeddings
	originalEmbeddings, err := predict.FetchTokenEmbeddings(predictor)
	if err != nil {
		return Trainer{}, errors.Wrapf(err, "unable to extract original token embeddings")
	}

	return Trainer{
		params:             params,
		in:                 in,
		allFilePaths:       in.Files,
		trainFilePaths:     train,
		validateFilePaths:  validate,
		originalEncoder:    predictor.GetEncoder(),
		originalEmbeddings: originalEmbeddings,
	}, nil
}

// Results ...
// TODO: once we can do the training in c/c++ we can change this struct
type Results struct {
	NewVocab   NewVocab
	NewEncoder *lexicalv0.FileEncoder

	TrainSamples    []WeightedSample
	ValidateSamples []WeightedSample
}

// Train ...
func (t Trainer) Train(kctx kitectx.Context) (Results, error) {
	newEncoder, mergeLog, err := t.extractNewVocab(kctx)
	if err != nil {
		return Results{}, errors.Wrapf(err, "unable to extract new vocab")
	}

	newVocab, err := InitializeNewVocab(t.originalEncoder, newEncoder, t.params.VocabInit, t.originalEmbeddings, mergeLog)
	if err != nil {
		return Results{}, errors.Wrapf(err, "unable to initialize new vocab")
	}

	train, validate := t.extractSamples(kctx, t.originalEncoder, newEncoder, newVocab)

	return Results{
		NewEncoder:      newEncoder,
		NewVocab:        newVocab,
		TrainSamples:    train,
		ValidateSamples: validate,
	}, nil
}

func (t Trainer) extractSamples(kctx kitectx.Context, originalEncoder, newEncoder *lexicalv0.FileEncoder, newVocab NewVocab) ([]WeightedSample, []WeightedSample) {
	kctx.CheckAbort()

	newIDs := make(map[int]bool, len(newVocab.NewIDs))
	for _, nid := range newVocab.NewIDs {
		newIDs[nid] = true
	}

	log.Println("building training samples...")
	allTrainSamples := RetrieveSamples(t.trainFilePaths, t.in.ContextSize, newEncoder, t.originalEncoder, newIDs, t.params.WeightedSampling)

	selectedTrainSamples := SelectSamples(allTrainSamples, t.params.TrainSampleRate, t.in.Seed)

	log.Println("building validation samples...")
	allValidateSamples := RetrieveSamples(t.validateFilePaths, t.in.ContextSize, newEncoder, originalEncoder, newIDs, t.params.WeightedSampling)

	r := rand.New(rand.NewSource(t.in.Seed))
	r.Shuffle(len(allValidateSamples), func(i, j int) {
		allValidateSamples[i], allValidateSamples[j] = allValidateSamples[j], allValidateSamples[i]
	})

	return selectedTrainSamples, allValidateSamples
}

func (t Trainer) extractNewVocab(kctx kitectx.Context) (*lexicalv0.FileEncoder, []bpe.MergedPair, error) {
	kctx.CheckAbort()

	builder := bpe.NewBuilderFromEncoder(t.originalEncoder.BPE)

	err := ExtractNewVocabEntries(kctx, t.allFilePaths, t.in.Language, builder, t.params.NumGo, t.params.VocabIters)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unable to extract new vocab")
	}

	encoder, err := lexicalv0.NewFileEncoderFromVocab(builder.Vocab(), t.in.Language)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unable to build new encoder")
	}

	return encoder, builder.MergeLog(), nil
}
