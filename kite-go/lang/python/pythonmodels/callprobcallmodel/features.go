package callprobcallmodel

import (
	"fmt"
	"math"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// NameAndWeight ...
type NameAndWeight struct {
	Name   string
	Weight float32
}

type processedPopularPattern struct {
	posArgCount  int
	keywordsArgs strSet
	frequency    float64
}

// String ...
func (n NameAndWeight) String() string {
	return fmt.Sprintf("%v: %v", n.Name, n.Weight)
}

// ContextualFeatures represents a subset of features that can be derived just from the source code.
type ContextualFeatures struct {
	NumVars int `json:"num_vars"` // number of variables in scope at the prediction site
}

func (c ContextualFeatures) vector() []float32 {
	weights := c.Weights()
	return []float32{weights[0].Weight, weights[1].Weight}
}

// Weights ...
func (c ContextualFeatures) Weights() []pythongraph.NameAndWeight {
	var numVars float32
	if c.NumVars > 0 {
		numVars = float32(math.Log(float64(c.NumVars)))
	}

	return []pythongraph.NameAndWeight{
		{Name: "bias", Weight: 1},
		{Name: "num_vars", Weight: numVars},
	}
}

type strSet map[string]struct{}

func (ss strSet) isSubset(parent strSet) bool {
	for k := range ss {
		if _, ok := parent[k]; !ok {
			return false
		}
	}
	return true
}

func (ss strSet) equal(other strSet) bool {
	return len(ss) == len(other) && ss.isSubset(other)
}

// CompFeatures represents features specific to each call completion.
type CompFeatures struct {
	PatternFreq           float64 `json:"pattern_freq"`
	Score                 float64 `json:"score"`
	NumArgs               int     `json:"num_args"`
	TypeMatchScore        float64 `json:"type_match_score"`
	TypesViolated         float64 `json:"types_violated"`
	NameRepeatedFunc      float64 `json:"name_repeated_func"`
	EffectiveArgs         float64 `json:"effective_args"`
	SubtokMatchScore      float64 `json:"subtok_match_score"`
	SubtoksViolated       float64 `json:"subtoks_violated"`
	Skip                  bool
	PlaceholderCount      float64 `json:"placeholder_count"`
	PlaceholderScopeRatio float64 `json:"placeholder_scope_ratio"`
}

// Weights ...
func (c CompFeatures) Weights() []pythongraph.NameAndWeight {
	return []pythongraph.NameAndWeight{
		{Name: "num_args", Weight: float32(c.NumArgs)},
		{Name: "pattern_freq", Weight: float32(c.PatternFreq)},
		{Name: "score", Weight: float32(c.Score)},
		{Name: "type_match_score", Weight: float32(c.TypeMatchScore)},
		{Name: "types_violated", Weight: float32(c.TypesViolated)},
		{Name: "name_repeated_func", Weight: float32(c.NameRepeatedFunc)},
		{Name: "effective_args", Weight: float32(c.EffectiveArgs)},
		{Name: "subtok_match_score", Weight: float32(c.SubtokMatchScore)},
		{Name: "subtoks_violated", Weight: float32(c.SubtoksViolated)},
		{Name: "placeholder_count", Weight: float32(c.PlaceholderCount)},
		{Name: "placeholder_scope_ratio", Weight: float32(c.PlaceholderScopeRatio)},
	}
}

func (c CompFeatures) vector() []float32 {
	weights := c.Weights()

	fs := make([]float32, 0, len(weights))
	for _, w := range weights {
		fs = append(fs, w.Weight)
	}
	return fs
}

// Features used for prediction
type Features struct {
	Contextual ContextualFeatures `json:"contextual"`
	Comp       []CompFeatures     `json:"comp"`
}

// TODO: for keyword arguments, we should use a different logic to compute the match.
// if the call arg matches any of the types in the kwarg and the freq is more than 1, then inPattern is 1
type callArg struct {
	name          string
	value         string
	isPlaceholder bool
}

// return slice of representations of the call arguments with name, value, and if it's a placeholder
func getCallArgs(args []pythongraph.PredictedCallArg) []callArg {
	var callArgs []callArg
	for _, arg := range args {
		if arg.Stop {
			break
		}
		callArgs = append(callArgs, callArg{
			name:          arg.Name,
			value:         arg.Value,
			isPlaceholder: arg.Value == pythongraph.PlaceholderPlaceholder,
		})
	}
	return callArgs
}

type argSpecs struct {
	exists         int
	numRequiredArg int
	varg           int
	kwarg          int
	spec           *pythonimports.ArgSpec
}

func countRequiredArgs(spec *pythonimports.ArgSpec) int {
	var numRequiredArg int
	for _, arg := range spec.NonReceiverArgs() {
		if !arg.Required() {
			break
		}
		numRequiredArg++
	}
	return numRequiredArg
}

func getArgSpec(rm pythonresource.Manager, sym pythonresource.Symbol) argSpecs {
	spec := rm.ArgSpec(sym)
	if spec == nil {
		return argSpecs{}
	}

	as := argSpecs{
		exists:         1,
		spec:           spec,
		numRequiredArg: countRequiredArgs(spec),
	}

	if spec.Vararg != "" {
		as.varg = 1
	}

	if spec.Kwarg != "" {
		as.kwarg = 1
	}

	return as
}

func isValidCall(comp pythongraph.PredictedCall, as argSpecs) bool {
	if as.spec == nil {
		return true
	}

	var required []string
	for _, arg := range as.spec.NonReceiverArgs() {
		// TODO: support keyword only args
		if !arg.Required() || arg.KeywordOnly {
			break
		}
		required = append(required, arg.Name)
	}

	argPresent := make(map[string]int)
	if comp.NumOrigArgs > 0 {
		for i := 0; i < comp.NumOrigArgs && i < len(required); i++ {
			argPresent[required[i]] = i - comp.NumOrigArgs
		}

		if comp.NumOrigArgs >= len(required) {
			required = nil
		} else {
			required = required[comp.NumOrigArgs:]
		}
	}

	for i, a := range comp.Args {
		name := a.Name
		if name == "" {
			if i >= len(as.spec.Args) {
				continue
			}
			name = as.spec.Args[i].Name
		}
		if _, ok := argPresent[name]; ok {
			// Arg duplicated, invalid call
			return false
		}
		argPresent[name] = i
	}

	requiredCount := len(required)
	var argOk int
	for _, a := range required {
		if pos, ok := argPresent[a]; ok {
			if pos < requiredCount {
				argOk++
			} else {
				// Required args should come first
				return false
			}
		}
	}
	if comp.PartialCall && argOk < requiredCount && argOk < len(comp.Args) {
		// That means we have some non required args already filled and we don't filled all required ones
		return false
	}

	if !comp.PartialCall && argOk < len(required) {
		return false
	}

	return true
}

type symbolInfo struct {
	sym  pythonresource.Symbol
	freq float64
}

type argInfo struct {
	isPlaceholder bool
	syms          []symbolInfo
	subtoks       []string
}

type argumentTable struct {
	positionals []argInfo
	keywords    map[string]argInfo
}

func newArgumentTable() argumentTable {
	return argumentTable{
		positionals: nil,
		keywords:    map[string]argInfo{},
	}
}

func (at *argumentTable) addArgument(name string, arg argInfo) {
	if name != "" {
		at.keywords[name] = arg
	} else {
		at.positionals = append(at.positionals, arg)
	}
}

func (at *argumentTable) size() int {
	return len(at.keywords) + len(at.positionals)
}

type subtokenTable struct {
	positionals []map[string]struct{}
	keywords    map[string]map[string]struct{}
}

// NoExprError is returned when we cannot find an expr at cursor
type NoExprError struct{}

// Error implements error.Error
func (e NoExprError) Error() string {
	return fmt.Sprintf("can't find a expr that has the cursor")
}

// NoArgSpecsError is returned for functions that we cannot find argspecs for
type NoArgSpecsError struct {
	variable string
}

// Error implements error.Error
func (e NoArgSpecsError) Error() string {
	return fmt.Sprintf("no spec for %s", e.variable)
}

// NoSymbolInTableError is a custom error type for handling this error until we fix the root cause.
type NoSymbolInTableError struct {
	variable string
}

func (e NoSymbolInTableError) Error() string {
	return fmt.Sprintf(fmt.Sprintf("can't find %v in symbol table", e.variable))
}

// NoSigStatsError is returned for functions that we could not find the symbol for
// TODO: figure out why this happens
type NoSigStatsError struct {
	sym pythonresource.Symbol
}

// Error implements error.Error
func (e NoSigStatsError) Error() string {
	return fmt.Sprintf("cant find sig stats for %v", e.sym)
}

// convert string values of argument of the call completion to pythonresource.Symbol and put them in a table keyed by postion or keyword name.
func typesForCallArguments(in Inputs, args []callArg, c pythongraph.PredictedCall) (argumentTable, error) {
	if len(args) == 0 {
		return argumentTable{}, nil
	}
	// find the symbol table from RAST
	var expr pythonast.Expr
	pythonast.Inspect(in.RAST.Root, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}
		if int64(n.Begin()) > in.Cursor || in.Cursor > int64(n.End()) {
			return false
		}
		// we may see deep non-exprs; e.g. the approxparser will generate a BadStmt exactly at the cursor in
		//   for x in y‸
		// representing the body of the for loop.
		if e, ok := n.(pythonast.Expr); ok {
			expr = e
		}
		return true
	})

	if pythonast.IsNil(expr) {
		return argumentTable{}, NoExprError{}
	}

	table, _ := in.RAST.TableAndScope(expr)
	if table == nil {
		return argumentTable{}, errors.Errorf("can't get table from RAST")
	}

	at := newArgumentTable()
	for _, arg := range args {
		argName := arg.name
		s := strings.TrimSpace(arg.value)
		// arg is a place holder we add it to the argument table but don't calculate the type
		if arg.isPlaceholder || s == "" {
			at.addArgument(argName, argInfo{isPlaceholder: true})
			continue
		}

		if !pythonscanner.IsValidIdent(s) {
			return argumentTable{}, errors.Errorf("getting an invalid python Ident:%v", s)
		}

		sym := table.Find(s)
		if sym == nil {
			return argumentTable{}, NoSymbolInTableError{variable: s}
		}

		ai := argInfo{
			subtoks: traindata.SplitNameLiteral(s),
		}

		if val := sym.Value; val != nil {
			valType := val.Type()
			valType = pythontype.Translate(kitectx.Background(), valType, in.RM)
			for _, v := range pythontype.Disjuncts(kitectx.Background(), valType) {
				ext, ok := v.(pythontype.External)
				if !ok {
					continue
				}
				ai.syms = append(ai.syms, symbolInfo{sym: ext.Symbol()})
			}
		}

		at.addArgument(argName, ai)
	}
	return at, nil
}

func subtoksForParameter(pe *editorapi.ParameterExample, target map[string]struct{}) {
	if pe == nil {
		return
	}

	addAll := func(items []string) {
		for _, s := range items {
			target[s] = struct{}{}
		}
	}

	if pe.Name != "" {
		addAll(traindata.SplitNameLiteral(pe.Name))
	}

	for _, t := range pe.Types {
		for _, e := range t.Examples {
			e = strings.Trim(e, "'\"")
			if pythonscanner.IsValidIdent(e) {
				addAll(traindata.SplitNameLiteral(e))
			}
		}
	}
}

// isStringLike checks if the symbol of a type is string-like
func isStringLike(s pythonresource.Symbol) bool {
	s = s.Canonical()
	if s.Distribution() != keytypes.BuiltinDistribution3 {
		return false
	}
	stringHashes := map[pythonimports.Hash]struct{}{
		pythonimports.PathHash([]byte("builtins.str")):  struct{}{},
		pythonimports.PathHash([]byte("typing.AnyStr")): struct{}{},
	}
	if _, ok := stringHashes[s.PathHash()]; !ok {
		return false
	}
	return true
}

// isArrayLike checks if the symbol of a type is array-like
func isArrayLike(s pythonresource.Symbol) bool {
	s = s.Canonical()
	if s.Distribution() != keytypes.BuiltinDistribution3 && s.Distribution() != keytypes.NumpyDistribution {
		return false
	}
	arrayHashes := map[pythonimports.Hash]struct{}{
		pythonimports.PathHash([]byte("builtins.list")):  struct{}{},
		pythonimports.PathHash([]byte("numpy.ndarray")):  struct{}{},
		pythonimports.PathHash([]byte("builtins.tuple")): struct{}{},
	}
	if _, ok := arrayHashes[s.PathHash()]; !ok {
		return false
	}
	return true
}

func ratioMatchedTypes(compValue argInfo, popValue pythonresource.SigStatArg, symMap symbolMap) float64 {
	if compValue.isPlaceholder {
		return 0.5
	}
	ratio := 1.0
	if popValue.Count > 0 {
		ratio = 1.0 / float64(popValue.Count)
	}
	var fracSymMatched float64
	for _, cSymbol := range compValue.syms {
		for _, sType := range popValue.Types {
			sSym, ok := symMap.getSymbol(sType)
			if !ok {
				continue
			}
			if cSymbol.sym.Equals(sSym) {
				fracSymMatched += float64(sType.Count) * ratio
				continue
			}

			if isStringLike(cSymbol.sym) && isStringLike(sSym) {
				fracSymMatched += float64(sType.Count) * ratio
				continue
			}

			if isArrayLike(cSymbol.sym) && isArrayLike(sSym) {
				fracSymMatched += float64(sType.Count) * ratio
				continue
			}
		}
	}
	return fracSymMatched
}

func ratioMatchedSubtoks(comp argInfo, subTokList map[string]struct{}) float64 {
	if comp.isPlaceholder {
		return 0.5
	}

	var matchCount int
	for _, ct := range comp.subtoks {
		var matched bool
		for pt := range subTokList {
			// we do a contains so that a completion like `req` counts as a match for `requests`
			if strings.Contains(pt, ct) || strings.Contains(ct, pt) {
				matched = true
				break
			}
		}

		if matched {
			matchCount++
		}
	}

	if len(comp.subtoks) == 0 {
		return 0
	}

	return float64(matchCount) / float64(len(comp.subtoks))
}

func subtokenMatch(args argumentTable, subtoks subtokenTable) (match, violated float64) {
	match = 1
	violated = 0
	argsCount := args.size()
	if argsCount == 0 {
		return match, violated
	}

	var subTokList map[string]struct{}
	for i, a := range args.positionals {
		if i >= len(subtoks.positionals) {
			// TODO check type for vararg
			continue
		}
		subTokList = subtoks.positionals[i]
		ratio := ratioMatchedSubtoks(a, subTokList)
		if ratio < .25 {
			match = 0
			violated++
		}
	}
	for n, a := range args.keywords {
		var ok bool
		subTokList, ok = subtoks.keywords[n]
		if !ok {
			// Unknown keyword name
			continue
		}
		ratio := ratioMatchedSubtoks(a, subTokList)
		if ratio < .25 {
			match = 0
			violated++
		}
	}
	violated = -violated / float64(args.size())
	return match, violated
}

type symbolMap map[string]pythonresource.Symbol

func (sm symbolMap) getSymbol(ti pythonresource.SigStatTypeInfo) (pythonresource.Symbol, bool) {
	sym, ok := sm[ti.GetSymKey()]
	return sym, ok
}

func buildSymbolMap(rm pythonresource.Manager, stats *pythonresource.SigStats) symbolMap {
	m := make(symbolMap)
	for _, arg := range stats.ArgsByName {
		for _, t := range arg.Types {
			key := t.GetSymKey()
			if _, ok := m[key]; !ok {
				newSym, err := rm.NewSymbol(t.Dist, pythonimports.NewDottedPath(t.Path))
				if err == nil {
					m[key] = newSym
				}
			}
		}
	}
	return m
}

func typeMatch(comp argumentTable, stats *pythonresource.SigStats, symMap symbolMap) (match, violated float64) {
	match = 1
	violated = 0
	if comp.size() == 0 {
		return match, violated
	}

	var as pythonresource.SigStatArg
	for i, a := range comp.positionals {
		if i >= len(stats.Positional) {
			// TODO check type for vararg
			continue
		}
		as = stats.Positional[i]
		ratio := ratioMatchedTypes(a, as, symMap)
		if ratio < .5 {
			match = 0
			violated++
		}
	}

	for n, a := range comp.keywords {
		var ok bool
		as, ok = stats.ArgsByName[n]
		if !ok {
			// Unknown keyword name
			continue
		}
		ratio := ratioMatchedTypes(a, as, symMap)
		if ratio < .5 {
			match = 0
			violated++
		}
	}
	violated = -violated / float64(comp.size())
	return match, violated
}

func numArgs(args []pythongraph.PredictedCallArg) int {
	var numArgs int
	for _, arg := range args {
		if arg.Stop {
			break
		}
		numArgs++
	}
	return numArgs
}

func frequency(call pythongraph.PredictedCall, popularPatterns []processedPopularPattern) float64 {
	partialCall := true
	if len(call.Args) > 0 && call.Args[len(call.Args)-1].Stop {
		partialCall = false
	}
	var posArgCount int
	keywordArgs := make(strSet)
	for _, arg := range call.Args {
		if arg.Name == "" {
			posArgCount++
		} else {
			keywordArgs[arg.Name] = struct{}{}
		}
	}

	var result float64
	for _, sig := range popularPatterns {
		if partialCall {
			if sig.posArgCount >= posArgCount && len(keywordArgs) == 0 {
				result += sig.frequency
				continue
			}
			if sig.posArgCount == posArgCount {
				if !keywordArgs.isSubset(sig.keywordsArgs) {
					continue
				}
				result += sig.frequency
			}
		} else {
			if sig.posArgCount != posArgCount {
				continue
			}
			if !keywordArgs.equal(sig.keywordsArgs) {
				continue
			}
			// There can be only one pattern matching exactly, we can return immediately
			return sig.frequency
		}
	}
	return result
}

func newSubtokenTable(popPatterns []*editorapi.Signature) subtokenTable {
	var table subtokenTable
	table.keywords = make(map[string]map[string]struct{})
	var maxPos int
	for _, sig := range popPatterns {
		if maxPos < len(sig.Args) {
			maxPos = len(sig.Args)
		}
	}
	table.positionals = make([]map[string]struct{}, maxPos)
	for i := 0; i < maxPos; i++ {
		table.positionals[i] = make(map[string]struct{})
	}
	for _, sig := range popPatterns {
		for i, arg := range sig.Args {
			subtoksForParameter(arg, table.positionals[i])
		}
		if sig.LanguageDetails.Python != nil {
			for _, arg := range sig.LanguageDetails.Python.Kwargs {
				n := arg.Name
				m, ok := table.keywords[n]
				if !ok {
					m = make(map[string]struct{})
					table.keywords[n] = m
				}
				subtoksForParameter(arg, m)
			}
		}
	}
	return table
}

// NewFeatures calculated from the inputs
func NewFeatures(in Inputs) (Features, error) {
	if len(in.CallComps) == 0 {
		return Features{}, errors.Errorf("no call completions")
	}

	aSpecs := getArgSpec(in.RM, in.Sym)
	if aSpecs.spec == nil {
		return Features{}, NoArgSpecsError{in.Sym.PathString()}
	}
	popularPatterns := in.RM.PopularSignatures(in.Sym)
	processedPopPatterns, avgArgCount := preparePopularPatterns(popularPatterns)
	sigStats := in.RM.SigStats(in.Sym)
	if sigStats == nil {
		return Features{}, NoSigStatsError{sym: in.Sym}
	}

	symMap := buildSymbolMap(in.RM, sigStats)
	subtokenMap := newSubtokenTable(popularPatterns)
	comp := make([]CompFeatures, 0, len(in.CallComps))
	for _, c := range in.CallComps {
		numArgs := numArgs(c.Args)

		if !isValidCall(c, aSpecs) {
			// invalid call so we just skip
			comp = append(comp, CompFeatures{Skip: true})
			continue
		}

		nScore := float64(c.Prob)

		callArgs := getCallArgs(c.Args)
		callArgsTypes, err := typesForCallArguments(in, callArgs, c)
		if err != nil {
			return Features{}, err
		}

		var sameArg float64
		// TODO: Might need to think more carefully about the case, inwhich there is less than 1 arg
		// get binary tag for if argument1 and argument2 of a specific completion is the same

	outer:
		for i := 0; i < len(callArgs)-1; i++ {
			for j := i + 1; j < len(callArgs); j++ {
				if strings.TrimSpace(callArgs[i].value) == strings.TrimSpace(callArgs[j].value) && !callArgs[i].isPlaceholder {
					sameArg = -1
					break outer
				}
			}
		}

		effectiveArgs := math.Abs(avgArgCount - float64(numArgs))
		effectiveFreq := frequency(c, processedPopPatterns)
		typeMatchScore, typeViolated := typeMatch(callArgsTypes, sigStats, symMap)
		effectiveFreq *= typeMatchScore
		subtokenMatchScore, subtokenViolated := subtokenMatch(callArgsTypes, subtokenMap)
		placeholderCount := countPlaceholder(c)
		placeholderScopeVarRatio := math.Log(1 + placeholderCount*float64(in.ScopeSize))

		comp = append(comp, CompFeatures{
			PatternFreq:           effectiveFreq,
			Score:                 nScore,
			NumArgs:               numArgs,
			NameRepeatedFunc:      10 * sameArg,
			EffectiveArgs:         effectiveArgs,
			SubtokMatchScore:      subtokenMatchScore,
			SubtoksViolated:       subtokenViolated,
			TypeMatchScore:        typeMatchScore,
			TypesViolated:         typeViolated,
			PlaceholderCount:      placeholderCount,
			PlaceholderScopeRatio: placeholderScopeVarRatio,
		})
	}

	return Features{
		Contextual: ContextualFeatures{
			NumVars: in.ScopeSize,
		},
		Comp: comp,
	}, nil
}

func countPlaceholder(call pythongraph.PredictedCall) float64 {
	result := float64(0)
	for _, a := range call.Args {
		if a.Placeholder() {
			result++
		}
	}
	return result
}

func preparePopularPatterns(signatures []*editorapi.Signature) ([]processedPopularPattern, float64) {
	result := make([]processedPopularPattern, 0, len(signatures))
	var averageArgCount float64
	for _, s := range signatures {
		averageArgCount += float64(len(s.Args))
		var keywords map[string]struct{}
		if s.LanguageDetails.Python != nil && len(s.LanguageDetails.Python.Kwargs) > 0 {
			averageArgCount += float64(len(s.LanguageDetails.Python.Kwargs))
			keywords = make(map[string]struct{}, len(s.LanguageDetails.Python.Kwargs))
			for _, ka := range s.LanguageDetails.Python.Kwargs {
				keywords[ka.Name] = struct{}{}
			}
		}
		result = append(result, processedPopularPattern{
			posArgCount:  len(s.Args),
			keywordsArgs: keywords,
			frequency:    s.Frequency,
		})
	}
	if len(signatures) > 0 {
		averageArgCount /= float64(len(signatures))
	}
	return result, averageArgCount
}

func (f Features) feeds() map[string]interface{} {
	compFeatures := make([][]float32, 0, len(f.Comp))
	for _, c := range f.Comp {
		compFeatures = append(compFeatures, c.vector())
	}

	return map[string]interface{}{
		"placeholders/contextual_features": [][]float32{f.Contextual.vector()},
		"placeholders/completion_features": compFeatures,
		"placeholders/sample_ids":          make([]int32, len(f.Comp)),
	}
}
