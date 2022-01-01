package pythongraph

// ExprTrainSample contains the information required for an infer expr train sample
type ExprTrainSample struct {
	ContextGraph    GraphFeed               `json:"context_graph"`
	InferName       NameModelFeed           `json:"infer_name"`
	InferProduction ProductionModelFeed     `json:"infer_production"`
	ExpansionGraph  ExpansionGraphTrainFeed `json:"expansion_graph"`
}

// ExprTrainInputs bundles the inputs required to build an expression training sample
type ExprTrainInputs struct {
	Call           *CallTrainInputs
	Attr           *AttributeTrainInputs
	AttrBase       *AttrBaseTrainInputs
	ArgType        *ArgTypeTrainInputs
	KwargName      *KwargNameTrainInputs
	ArgPlaceholder *ArgPlaceholderTrainInputs
}

// NewExprTrainSample builds a new expr training sample
func NewExprTrainSample(config TrainConfig, params TrainParams, ins ...ExprTrainInputs) (*ExprTrainSample, error) {
	var prods []*InferProductionSample
	var names []*InferNameSample
	var errs []TrainSampleErr
	for _, in := range ins {
		switch {
		case in.Call != nil:
			call, err := NewCallTrainSample(config, params, *in.Call)
			if err != nil {
				errs = append(errs, TrainSampleErr{
					Symbol:          in.Call.Symbol,
					Hash:            in.Call.Hash,
					Err:             err,
					ExprSubTaskType: InferCallTask,
				})
				continue
			}
			names = append(names, call)
		case in.Attr != nil:
			attr, err := NewAttributeTrainSample(config, params, *in.Attr)
			if err != nil {
				errs = append(errs, TrainSampleErr{
					Symbol:          in.Attr.Symbol,
					Hash:            in.Attr.Hash,
					Err:             err,
					ExprSubTaskType: InferAttrTask,
				})
				continue
			}
			prods = append(prods, attr)
		case in.AttrBase != nil:
			base, err := NewAttrBaseTrainSample(config, params, *in.AttrBase)
			if err != nil {
				errs = append(errs, TrainSampleErr{
					Symbol:          in.AttrBase.Symbol,
					Hash:            in.AttrBase.Hash,
					Err:             err,
					ExprSubTaskType: InferAttrBaseTask,
				})
				continue
			}
			names = append(names, base)
		case in.ArgType != nil:
			argType, err := NewArgTypeTrainSample(config, params, *in.ArgType)
			if err != nil {
				errs = append(errs, TrainSampleErr{
					Symbol:          in.ArgType.Symbol,
					Hash:            in.ArgType.Hash,
					Err:             err,
					ExprSubTaskType: InferArgTypeTask,
				})
				continue
			}
			prods = append(prods, argType)
		case in.KwargName != nil:
			kwargName, err := NewKwargNameTrainSample(config, params, *in.KwargName)
			if err != nil {
				errs = append(errs, TrainSampleErr{
					Symbol:          in.KwargName.Symbol,
					Hash:            in.KwargName.Hash,
					Err:             err,
					ExprSubTaskType: InferKwargNameTask,
				})
				continue
			}
			prods = append(prods, kwargName)
		case in.ArgPlaceholder != nil:
			argPlaceholder, err := NewArgPlaceholderTrainSample(config, params, *in.ArgPlaceholder)
			if err != nil {
				errs = append(errs, TrainSampleErr{
					Symbol:          in.ArgPlaceholder.Symbol,
					Hash:            in.ArgPlaceholder.Hash,
					Err:             err,
					ExprSubTaskType: InferArgPlaceholderTask,
				})
				continue
			}
			prods = append(prods, argPlaceholder)

		default:
			panic("empty infer expr training input")
		}
	}

	if len(prods) == 0 && len(names) == 0 {
		return nil, TrainSampleErrs(errs)
	}

	// NOTE: make sure to initialize all array fields to avoid
	// them getting serialized as nil
	sample := &ExprTrainSample{
		ContextGraph: GraphFeed{
			Edges: make(EdgeFeed),
		},
		ExpansionGraph: ExpansionGraphTrainFeed{
			ExpansionGraphBaseFeed: ExpansionGraphBaseFeed{
				Edges: make(EdgeFeed),
			},
		},
		InferName:       newNameModelFeed(),
		InferProduction: newEmptyProductionModelFeed(),
	}

	var contextOffset, expansionOffset NodeID
	func() {
		var sampleID int
		var varOffset VariableID
		for _, name := range names {
			sample.ContextGraph = sample.ContextGraph.append(name.ContextGraph, contextOffset)
			sample.ExpansionGraph = sample.ExpansionGraph.append(name.ExpansionGraph, contextOffset, expansionOffset)

			sample.InferName = sample.InferName.append(name.Name, sampleID, expansionOffset, varOffset)

			sampleID++

			contextOffset += NodeID(name.ContextGraph.NumNodes())
			expansionOffset += NodeID(name.ExpansionGraph.NumNodes())
			varOffset += VariableID(name.Name.Names.NumVariables())
		}
	}()

	func() {
		var sampleID int
		for _, prod := range prods {
			sample.ContextGraph = sample.ContextGraph.append(prod.ContextGraph, contextOffset)
			sample.ExpansionGraph = sample.ExpansionGraph.append(prod.ExpansionGraph, contextOffset, expansionOffset)

			sample.InferProduction = sample.InferProduction.append(prod.Production, sampleID, expansionOffset)

			sampleID++

			contextOffset += NodeID(prod.ContextGraph.NumNodes())
			expansionOffset += NodeID(prod.ExpansionGraph.NumNodes())
		}
	}()

	return sample, TrainSampleErrs(errs)
}
