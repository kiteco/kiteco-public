package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-errors/errors"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

type inferAttrReq struct {
	Symbols traindata.SymbolDist `json:"symbols"`
	Parents map[string]string    `json:"parents"`
	// proportion of the samples in the batch to use for this task
	BatchProportion float64 `json:"batch_proportion"`
}

func (r inferAttrReq) Valid() error {
	if r.BatchProportion < 0 {
		return fmt.Errorf("batch proportion %v must be atleast 0", r.BatchProportion)
	}
	return nil
}

type inferAttrBaseReq struct {
	Symbols traindata.SymbolDist `json:"symbols"`
	// proportion of the samples in the batch to use for this task
	BatchProportion float64 `json:"batch_proportion"`
}

func (r inferAttrBaseReq) Valid() error {
	if r.BatchProportion < 0 {
		return fmt.Errorf("batch proportion %v must be atleast 0", r.BatchProportion)
	}
	return nil
}

type inferCallReq struct {
	Symbols         traindata.SymbolDist `json:"symbols"`
	BatchProportion float64              `json:"batch_proportion"`
}

type inferArgTypeReq struct {
	Symbols         traindata.SymbolDist `json:"symbols"`
	BatchProportion float64              `json:"batch_proportion"`
}

func (r inferArgTypeReq) Valid() error {
	if r.BatchProportion < 0 {
		return fmt.Errorf("batch proportion %v must be atleast 0", r.BatchProportion)
	}
	return nil
}

type inferKwargReq struct {
	Symbols         traindata.SymbolDist `json:"symbols"`
	Keywords        map[string][]string  `json:"keywords"`
	BatchProportion float64              `json:"batch_proportion"`
}

func (r inferKwargReq) Valid() error {
	if r.BatchProportion < 0 {
		return fmt.Errorf("batch proportion %v must be atleast 0", r.BatchProportion)
	}
	return nil
}

func (r inferCallReq) Valid() error {
	if r.BatchProportion < 0 {
		return fmt.Errorf("batch proportion %v must be atleast 0", r.BatchProportion)
	}

	return nil
}

type inferArgPlaceholderReq struct {
	Symbols         traindata.SymbolDist `json:"symbols"`
	BatchProportion float64              `json:"batch_proportion"`
}

func (r inferArgPlaceholderReq) Valid() error {
	if r.BatchProportion < 0 {
		return fmt.Errorf("batch proportion %v must be atleast 0", r.BatchProportion)
	}
	return nil
}

type inferExprReq struct {
	Call           inferCallReq           `json:"call"`
	Attr           inferAttrReq           `json:"attr"`
	AttrBase       inferAttrBaseReq       `json:"attr_base"`
	ArgType        inferArgTypeReq        `json:"arg_type"`
	KwargName      inferKwargReq          `json:"kwarg_name"`
	ArgPlaceholder inferArgPlaceholderReq `json:"arg_placeholder"`
	MaxSamples     uint32                 `json:"max_samples"`
}

func (e inferExprReq) Valid() error {
	if err := e.Call.Valid(); err != nil {
		return fmt.Errorf("invalid call: %v", err)
	}
	if err := e.Attr.Valid(); err != nil {
		return fmt.Errorf("invalid attr: %v", err)
	}
	if err := e.AttrBase.Valid(); err != nil {
		return fmt.Errorf("invalid attr base: %v", err)
	}
	if err := e.ArgType.Valid(); err != nil {
		return fmt.Errorf("invalid arg type: %v", err)
	}
	if err := e.KwargName.Valid(); err != nil {
		return fmt.Errorf("invalid kwarg name: %v", err)
	}
	if err := e.ArgPlaceholder.Valid(); err != nil {
		return fmt.Errorf("invalid arg placeholder value: %v", err)
	}

	var numNonZero uint32
	if e.Call.BatchProportion > 0 {
		numNonZero++
	}
	if e.Attr.BatchProportion > 0 {
		numNonZero++
	}
	if e.AttrBase.BatchProportion > 0 {
		numNonZero++
	}
	if e.ArgType.BatchProportion > 0 {
		numNonZero++
	}
	if e.KwargName.BatchProportion > 0 {
		numNonZero++
	}
	if e.ArgPlaceholder.BatchProportion > 0 {
		numNonZero++
	}

	if numNonZero == 0 {
		return fmt.Errorf("must include samples for at least one task")
	}

	if e.MaxSamples < numNonZero {
		return fmt.Errorf("have %v tasks, must include atleast this many samples, got %d", numNonZero, e.MaxSamples)
	}

	return nil
}

type sessionRequest struct {
	Session           int                         `json:"session"`
	RandomSeed        *int64                      `json:"random_seed"`
	Partition         interval                    `json:"partition"`
	MaxHops           uint64                      `json:"max_hops"`
	NumBatches        uint64                      `json:"num_batches"`
	Config            pythongraph.GraphFeedConfig `json:"config"`
	NameSubtokenIndex traindata.SubtokenIndex     `json:"name_subtoken_index"`
	TypeSubtokenIndex traindata.SubtokenIndex     `json:"type_subtoken_index"`
	ProductionIndex   traindata.ProductionIndex   `json:"production_index"`
	Expr              *inferExprReq               `json:"expr"`
}

func (r sessionRequest) Valid() error {
	if err := r.Partition.Valid(); err != nil {
		return err
	}

	if r.RandomSeed == nil {
		return fmt.Errorf("need to specify random seed")
	}

	if len(r.NameSubtokenIndex) == 0 {
		return fmt.Errorf("need to specify a name subtoken index")
	}

	if len(r.TypeSubtokenIndex) == 0 {
		return fmt.Errorf("need to specify a type subtoken index")
	}

	if len(r.ProductionIndex.Indices) == 0 {
		return fmt.Errorf("need production index")
	}

	if err := r.Config.Valid(); err != nil {
		return fmt.Errorf("invalid graph feed config: %v", err)
	}

	if err := r.Expr.Valid(); err != nil {
		return fmt.Errorf("invalid expr: %v", err)
	}

	return nil
}

func (a *app) handleSession(w http.ResponseWriter, r *http.Request) {
	var req sessionRequest
	if err := a.decode(r.Body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var session *session
	if req.Session == 0 {
		if err := req.Valid(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var err error
		session, err = a.buildNewExprSession(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("error building session: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		// non-zero session ID: get an existing session
		session = a.sessions.GetSession(sessionID(req.Session))
		if session == nil {
			http.Error(w, fmt.Sprintf("session not found for ID: %d", req.Session), http.StatusNotFound)
			return
		}
	}

	start := time.Now()

	batch, err := session.GetBatch()
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting next batch: %v", err), http.StatusInternalServerError)
		return
	}
	getSessionBatchDuration.RecordDuration(time.Since(start))

	start = time.Now()
	buf, err := json.Marshal(sessionResponse{
		Session: int(session.id),
		Samples: batch,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshaling response: %v", err), http.StatusInternalServerError)
		return
	}
	encodeSessionBatchDuration.RecordDuration(time.Since(start))

	start = time.Now()
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
	transferSessionBatchDuration.RecordDuration(time.Since(start))
}

func (a *app) buildNewExprSession(req sessionRequest) (*session, error) {
	call, err := newInferCallFeeder(&req.Expr.Call, *req.RandomSeed, req.Partition, a.res)
	if err != nil {
		return nil, err
	}

	attr, parents, err := newInferAttrFeeder(&req.Expr.Attr, *req.RandomSeed, req.Partition, a.res)
	if err != nil {
		return nil, err
	}

	attrBase, err := newInferAttrBaseFeeder(&req.Expr.AttrBase, *req.RandomSeed, req.Partition, a.res)
	if err != nil {
		return nil, err
	}

	argType, err := newInferArgTypeFeeder(&req.Expr.ArgType, *req.RandomSeed, req.Partition, a.res)
	if err != nil {
		return nil, err
	}

	kwargName, err := newInferKwargNameFeeder(&req.Expr.KwargName, *req.RandomSeed, req.Partition, a.res)
	if err != nil {
		return nil, err
	}

	argPlaceholder, err := newArgPlaceholderFeeder(&req.Expr.ArgPlaceholder, *req.RandomSeed, req.Partition, a.res)
	if err != nil {
		return nil, err
	}

	// grab parameters we need from req since we do not want to close over it
	// since then req will not be GCd

	nsti := req.NameSubtokenIndex
	tsti := req.TypeSubtokenIndex
	pi := req.ProductionIndex
	conf := req.Config
	maxHops := req.MaxHops

	const numTasks = 6
	feeders := [numTasks]*feeder{
		call,
		attr,
		attrBase,
		argType,
		kwargName,
		argPlaceholder,
	}

	total := req.Expr.Attr.BatchProportion + req.Expr.AttrBase.BatchProportion +
		req.Expr.Call.BatchProportion + req.Expr.ArgType.BatchProportion +
		req.Expr.KwargName.BatchProportion + req.Expr.ArgPlaceholder.BatchProportion

	sz := float64(req.Expr.MaxSamples)

	samplesPerBatch := [numTasks]uint64{
		uint64(math.Ceil(sz * req.Expr.Call.BatchProportion / total)),
		uint64(math.Ceil(sz * req.Expr.Attr.BatchProportion / total)),
		uint64(math.Ceil(sz * req.Expr.AttrBase.BatchProportion / total)),
		uint64(math.Ceil(sz * req.Expr.ArgType.BatchProportion / total)),
		uint64(math.Ceil(sz * req.Expr.KwargName.BatchProportion / total)),
		uint64(math.Ceil(sz * req.Expr.ArgPlaceholder.BatchProportion / total)),
	}

	build := func() (_ *sample, err error) {
		defer func() {
			if r := recover(); r != nil {
				// add strack trace, 1 level up so we skip the defered function frame
				err = errors.Wrap(r, 1)
				log.Println("panic while building batch:", err)
				log.Println(err.(*errors.Error).ErrorStack())
			}
		}()

		var exprIns []pythongraph.ExprTrainInputs
		for i := 0; i < numTasks; i++ {
			feeder := feeders[i]
			if feeder == nil {
				// empty task
				continue
			}

			seeds, err := getSeeds(feeder, samplesPerBatch[i])
			if err != nil {
				// TODO: log and continue?
				return nil, err
			}

			ins, bad := getBatchInputs(a.res, seeds...)

			for _, b := range bad {
				feeder.Invalidate(b.Symbol, b.Hash)
			}

			for _, in := range ins {
				var exprIn pythongraph.ExprTrainInputs
				switch i {
				case 0:
					exprIn = pythongraph.ExprTrainInputs{
						Call: &pythongraph.CallTrainInputs{
							Hash:   in.Seed.Hash,
							Symbol: in.Seed.Symbol,
							Inputs: in.In,
						},
					}
				case 1:
					exprIn = pythongraph.ExprTrainInputs{
						Attr: &pythongraph.AttributeTrainInputs{
							Hash:           in.Seed.Hash,
							Symbol:         in.Seed.Symbol,
							Inputs:         in.In,
							Parent:         parents[in.Seed.Symbol.PathString()],
							CanonicalToSym: attr.canonicalToSymbols,
						},
					}
				case 2:
					exprIn = pythongraph.ExprTrainInputs{
						AttrBase: &pythongraph.AttrBaseTrainInputs{
							Hash:   in.Seed.Hash,
							Symbol: in.Seed.Symbol,
							Inputs: in.In,
						},
					}
				case 3:
					exprIn = pythongraph.ExprTrainInputs{
						ArgType: &pythongraph.ArgTypeTrainInputs{
							Hash:   in.Seed.Hash,
							Symbol: in.Seed.Symbol,
							Inputs: in.In,
						},
					}
				case 4:
					exprIn = pythongraph.ExprTrainInputs{
						KwargName: &pythongraph.KwargNameTrainInputs{
							Hash:     in.Seed.Hash,
							Symbol:   in.Seed.Symbol,
							Inputs:   in.In,
							Keywords: req.Expr.KwargName.Keywords[in.Seed.Symbol.PathString()],
						},
					}
				case 5:
					exprIn = pythongraph.ExprTrainInputs{
						ArgPlaceholder: &pythongraph.ArgPlaceholderTrainInputs{
							Hash:   in.Seed.Hash,
							Symbol: in.Seed.Symbol,
							Inputs: in.In,
						},
					}
				default:
					panic(fmt.Sprintf("expected tasks to be between 0 and %d got %d", numTasks-1, i))
				}
				exprIns = append(exprIns, exprIn)
			}
		}

		config := pythongraph.TrainConfig{
			Graph:        conf,
			MaxHops:      int(maxHops),
			NumCorrupted: 5,
		}

		params := pythongraph.TrainParams{
			Rand: rand.New(rand.NewSource(call.rand.Int63())),
			ModelMeta: pythongraph.ModelMeta{
				NameSubtokenIndex: nsti,
				TypeSubtokenIndex: tsti,
				ProductionIndex:   pi,
			},
		}

		exprGraphsPerSample.Add(int64(len(exprIns)))

		start := time.Now()
		s, err := pythongraph.NewExprTrainSample(config, params, exprIns...)
		exprTrainSampleDuration.RecordDuration(time.Since(start))

		for _, b := range badSeedsFromErr(err) {
			switch b.ExprSubTaskType {
			case pythongraph.InferAttrBaseTask:
				attrBase.Invalidate(b.Symbol, b.Hash)
			case pythongraph.InferAttrTask:
				attr.Invalidate(b.Symbol, b.Hash)
			case pythongraph.InferCallTask:
				call.Invalidate(b.Symbol, b.Hash)
			case pythongraph.InferArgTypeTask:
				argType.Invalidate(b.Symbol, b.Hash)
			case pythongraph.InferKwargNameTask:
				kwargName.Invalidate(b.Symbol, b.Hash)
			case pythongraph.InferArgPlaceholderTask:
				argPlaceholder.Invalidate(b.Symbol, b.Hash)
			}
		}

		if s == nil {
			return nil, fmt.Errorf("unable to build sample: %v", err)
		}

		return &sample{
			Data: trainData{
				Expr: s,
			},
		}, nil
	}

	builder := newBuilder(int(req.NumBatches), build)

	return a.sessions.CreateSession(builder)
}

func newArgPlaceholderFeeder(req *inferArgPlaceholderReq, seed int64, partition interval, res *resources) (*feeder, error) {
	if req.BatchProportion == 0 {
		return nil, nil
	}

	if err := req.Valid(); err != nil {
		return nil, err
	}

	dist, err := newDistribution(req.Symbols, seed)
	if err != nil {
		return nil, fmt.Errorf("failed to make symbol distribution: %v", err)
	}

	feeder, err := newFeeder(dist, seed, pythoncode.SymbolContextCallFunc, partition, res)
	if err != nil {
		return nil, fmt.Errorf("error creating arg placeholder feeder: %v", err)
	}

	return feeder, nil
}

func newInferCallFeeder(req *inferCallReq, seed int64, partition interval, res *resources) (*feeder, error) {
	if req.BatchProportion == 0 {
		return nil, nil
	}

	if err := req.Valid(); err != nil {
		return nil, err
	}

	dist, err := newDistribution(req.Symbols, seed)
	if err != nil {
		return nil, fmt.Errorf("failed to make symbol distribution: %v", err)
	}

	feeder, err := newFeeder(dist, seed, pythoncode.SymbolContextCallFunc, partition, res)
	if err != nil {
		return nil, fmt.Errorf("error creating call feeder: %v", err)
	}

	return feeder, nil
}

func newInferAttrFeeder(req *inferAttrReq, seed int64, partition interval, res *resources) (*feeder, map[string]pythonresource.Symbol, error) {
	if req.BatchProportion == 0 {
		return nil, nil, nil
	}

	dist, err := newDistribution(req.Symbols, seed)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to make symbol distribution: %v", err)
	}

	feeder, err := newFeeder(dist, seed, pythoncode.SymbolContextAttribute, partition, res)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating attr feeder: %v", err)
	}

	parents := make(map[string]pythonresource.Symbol)
	for child, parent := range req.Parents {
		sym, err := getSymbol(parent, res.rm)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to find parent symbol: %v", err)
		}
		parents[child] = sym
	}

	return feeder, parents, nil
}

func newInferArgTypeFeeder(req *inferArgTypeReq, seed int64, partition interval, res *resources) (*feeder, error) {
	dist, err := newDistribution(req.Symbols, seed)
	if err != nil {
		return nil, fmt.Errorf("failed to make symbol distribution: %v", err)
	}

	feeder, err := newFeeder(dist, seed, pythoncode.SymbolContextCallFunc, partition, res)
	if err != nil {
		return nil, fmt.Errorf("error creating feeder: %v", err)
	}

	return feeder, nil
}

func newInferKwargNameFeeder(req *inferKwargReq, seed int64, partition interval, res *resources) (*feeder, error) {
	if req.BatchProportion == 0 {
		return nil, nil
	}

	dist, err := newDistribution(req.Symbols, seed)
	if err != nil {
		return nil, fmt.Errorf("failed to make symbol distribution: %v", err)
	}

	feeder, err := newFeeder(dist, seed, pythoncode.SymbolContextCallFunc, partition, res)
	if err != nil {
		return nil, fmt.Errorf("error creating feeder: %v", err)
	}

	return feeder, nil
}

func newInferAttrBaseFeeder(req *inferAttrBaseReq, seed int64, partition interval, res *resources) (*feeder, error) {
	if req.BatchProportion == 0 {
		return nil, nil
	}

	dist, err := newDistribution(req.Symbols, seed)
	if err != nil {
		return nil, fmt.Errorf("failed to make symbol distribution: %v", err)
	}

	feeder, err := newFeeder(dist, seed, pythoncode.SymbolContextName, partition, res)
	if err != nil {
		return nil, fmt.Errorf("error creating attr base feeder: %v", err)
	}
	return feeder, nil
}

type seedSrcBuildFn func(sampleSeed, []byte) (*sample, error)

func newBuildFunc(store *codeStore, fn seedSrcBuildFn, feeder *feeder) buildFunc {
	return func() (_ *sample, err error) {
		seed, err := feeder.Next()
		if err != nil {
			return nil, err
		}

		src, err := store.SourceFor(seed.Hash)
		if err != nil {
			return nil, err
		}

		if len(src) > maxFileBytes {
			return nil, fmt.Errorf("file too large with %d bytes", len(src))
		}

		defer func() {
			if r := recover(); r != nil {
				// add strack trace, 1 level up so we skip the defered function frame
				err = errors.Wrap(r, 1)
				log.Printf("panic while processing seed: %s %s %d:",
					seed.Hash, seed.Symbol.PathString(), seed.Random)
				log.Println(err)
				log.Println(err.(*errors.Error).ErrorStack())
			}
		}()

		start := time.Now()
		sample, err := fn(seed, src)
		buildDuration := time.Since(start)

		switch {
		case err != nil:
			feeder.Invalidate(seed.Symbol, seed.Hash)
			return nil, err
		case buildDuration > maxSampleBuildDuration:
			// Kind of nasty since this means that
			// we have already built the sample, but this
			// ensures that all symbol contexts share the same timeout logic.
			// TODO: update all training sample construction methods to take a kitectx
			// in and use that for timeouts.
			feeder.Invalidate(seed.Symbol, seed.Hash)
			return nil, fmt.Errorf("sample took to long to build")
		default:
			return sample, nil
		}
	}
}

type badSeed struct {
	ExprSubTaskType pythongraph.ExprSubTaskType
	Hash            string
	Symbol          pythonresource.Symbol
}

func badSeedsFromErr(err error) []badSeed {
	if se, ok := err.(pythongraph.TrainSampleErrs); ok {
		var bad []badSeed
		for _, e := range se {
			bad = append(bad, badSeed{
				ExprSubTaskType: e.ExprSubTaskType,
				Hash:            e.Hash,
				Symbol:          e.Symbol,
			})
		}
		return bad
	}
	return nil
}

type inputAndSeed struct {
	Seed sampleSeed
	Src  []byte
	In   pythongraph.Inputs
}

func getBatchInputs(res *resources, seeds ...sampleSeed) ([]inputAndSeed, []badSeed) {
	var ins []inputAndSeed
	var bad []badSeed
	for _, seed := range seeds {
		src, err := res.store.SourceFor(seed.Hash)
		if err != nil {
			bad = append(bad, badSeed{
				Hash:   seed.Hash,
				Symbol: seed.Symbol,
			})
			continue
		}

		if len(src) > maxFileBytes {
			bad = append(bad, badSeed{
				Hash:   seed.Hash,
				Symbol: seed.Symbol,
			})
			continue
		}

		in, err := getInputs(src, res)
		if err != nil {
			bad = append(bad, badSeed{
				Hash:   seed.Hash,
				Symbol: seed.Symbol,
			})
			continue
		}

		ins = append(ins, inputAndSeed{
			Seed: seed,
			Src:  src,
			In:   in,
		})
	}
	return ins, bad
}

func getSeeds(feeder *feeder, numSeeds uint64) ([]sampleSeed, error) {
	seeds := make([]sampleSeed, 0, numSeeds)
	for len(seeds) < int(numSeeds) {
		var seed sampleSeed
		for i := 0; i < maxFailedSeeds; i++ {
			var err error
			seed, err = feeder.Next()
			if err != nil {
				log.Println(err)
				continue
			}
			break
		}

		if seed.Hash == "" {
			return nil, fmt.Errorf("unable to get a seed after %d tries", maxFailedSeeds)
		}

		seeds = append(seeds, seed)
	}

	return seeds, nil
}
