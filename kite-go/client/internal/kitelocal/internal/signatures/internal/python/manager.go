package python

import (
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/calls"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/errors"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

// Manager for python signature experience.
type Manager struct {
	funcName   string
	callee     *editorapi.ValueExt
	signatures []*editorapi.Signature
	function   *editorapi.FunctionDetails
	start      int64
	hash       string
	devMode    bool
}

// NewManager returns a new python specific signature manager.
// Returns an error if callee does not contain function details, or if we are unable to find
// the start of the arguments for the call that the user's cursor is under.
func NewManager(callee *editorapi.ValueExt, sigs []*editorapi.Signature, funcName, filename, src string, cursor int64, devMode bool) (*Manager, error) {
	// make sure we can find the start of the arguments
	start := findArgsStart(src, cursor)
	if start < 0 {
		// TODO(juan): better way to structure this, this should not happen.
		errTypes.HitAndAdd("Unable to find call start")
		return nil, fmt.Errorf("unable to find call start")
	}

	// make sure we have a function set
	function := callee.Details.Function
	if function == nil && callee.Details.Type != nil && callee.Details.Type.LanguageDetails.Python != nil {
		function = callee.Details.Type.LanguageDetails.Python.Constructor
	}

	if function == nil {
		errTypes.HitAndAdd("No function set")
		return nil, fmt.Errorf("no function set")
	}

	return &Manager{
		funcName:   funcName,
		callee:     callee,
		signatures: sigs,
		function:   function,
		start:      start,
		hash:       fmt.Sprintf("%x", md5.Sum([]byte(src[:start]))),
		devMode:    devMode,
	}, nil
}

// Fetch returns true if a new callee must be fetched for src and cursor.
// This returns true in the case that the hash of the src contents from the beginning of the file to the start of the arguments has changed.
// This returns an error in the case that we were unable to find the start of the arguments for the call.
// SEE: Design decisions in README.md
func (m *Manager) Fetch(src string, cursor int64) (bool, error) {
	start := findArgsStart(src, cursor)
	if start < 0 {
		fetchBreakdown.HitAndAdd("unable to find start of arguments")
		return false, fmt.Errorf("unable to find start of arguments")
	}

	if m.hash != fmt.Sprintf("%x", md5.Sum([]byte(src[:start]))) {
		fetchBreakdown.HitAndAdd("fetch")
		return true, nil
	}

	fetchBreakdown.HitAndAdd("no fetch")
	return false, nil
}

// Handle the response for the specified source contents
// and cursor position.
// Returns the appropriate response or an http error code and error.
func (m *Manager) Handle(src string, cursor int64) *editorapi.SignatureResponse {
	// parse
	args, err := parse(src, m.start, cursor)
	var param parameter
	if err == nil {
		// find argument user's cursor is on
		arg := findArg(args, cursor)

		// find the parameter corresponding to the current argument
		param = findParam(m.function, arg)
	} else {
		errTypes.HitAndAdd("Parse error")
		// this ensures that we keep displaying signatures after
		// a parse error
		// TODO(juan): https://github.com/kiteco/kiteco/issues/5050
		param = parameter{Idx: -1}
	}

	return &editorapi.SignatureResponse{
		Language: lang.Python.Name(),
		Calls: []*editorapi.Call{
			&editorapi.Call{
				FuncName:   m.funcName,
				Callee:     m.callee,
				Signatures: m.signatures,
				ArgIndex:   param.Idx,
				LanguageDetails: editorapi.LanguageCallDetails{
					Python: &editorapi.PythonCallDetails{
						InKwargs: param.InKwargs,
					},
				},
			},
		},
	}
}

// arguments for a call.
type arguments struct {
	Args   []*pythonast.Argument
	Vararg pythonast.Expr
	Kwarg  pythonast.Expr
	Commas []*pythonscanner.Word
}

// argument for tracking the current argument
// that the user's cursor is on.
type argument struct {
	// Idx of the argument in the current call.
	Idx int
	// NumPositional is the number of positional arguments passed strictly before the argument containing the cursor
	NumPositional int
	// UsedKwargs is the set of keyword argument names passed strictly before the argument containing the cursor
	UsedKwargs map[string]struct{}
	// Name is the name the user typed before an "=";
	// if the user has not typed an "=", this is the name the user has typed;
	// if the user has typed a non-name expression, this is empty
	Name string
	// InKwarg is true if the user has typed an "=" in any argument up to the cursor
	InKwarg bool
}

func findArg(args arguments, cursor int64) argument {
	res := argument{UsedKwargs: make(map[string]struct{})}

	for _, comma := range args.Commas {
		if cursor < int64(comma.End) {
			break // assume commas are in order
		}

		if res.Idx < len(args.Args) {
			if name := args.Args[res.Idx].Name; !pythonast.IsNil(name) {
				if name, ok := name.(*pythonast.NameExpr); ok {
					res.UsedKwargs[name.Ident.Literal] = struct{}{}
				}
				res.InKwarg = true
			}
		}
		// otherwise, the user is typing immediately after a comma, in which case len(args.Args) == len(args.Commas)
		// or something went terribly wrong

		if !res.InKwarg {
			res.NumPositional++
		}

		res.Idx++
	}

	if res.Idx < len(args.Args) {
		if name := args.Args[res.Idx].Name; !pythonast.IsNil(name) {
			res.InKwarg = true
		}
	}

	if res.Idx >= len(args.Args) {
		// the user is typing the first argument, and there are 0 args
		// or the user is typing immediately after a comma, in which case len(args.Args) == len(args.Commas)
		// or something went terribly wrong
		return res
	}

	name, _ := args.Args[res.Idx].Name.(*pythonast.NameExpr)
	if name == nil {
		name, _ = args.Args[res.Idx].Value.(*pythonast.NameExpr)
	}
	if name != nil {
		res.Name = name.Ident.Literal
	}

	return res
}

// parameter for tracking the current parameter
// that the user is referring to.
type parameter struct {
	Idx int
	// InKwargs is true if the matched parameter is from function.LanguageDetails.Python.KwargParameters (i.e. a suggestion matching *kwargs)
	InKwargs bool
}

// findByName searches the arguments, finding the shortest parameter matching the name
// (shortest is best: if fo, foo are both matches for typed "fo", fo is better)
func findByName(function *editorapi.FunctionDetails, arg argument) parameter {
	numPositional := arg.NumPositional
	if numPositional > len(function.Parameters) {
		numPositional = len(function.Parameters)
	}

	match := parameter{Idx: -1}
	matchLen := -1

	// For local code functions with **kwargs, the index will frequently be rebuilt to contain what the user just typed in KwargParameters
	// so we have to *first* check Parameters, then KwargParameters (to avoid almost always choosing the exact match).
	// This means if there's a better match in KwargParameters than in Parameters, we won't select it, which is a pity TODO(naman)

	// note that explicit (non-*args) positional parameters may also be passed by name
	for idx, param := range function.Parameters[numPositional:] {
		if _, used := arg.UsedKwargs[param.Name]; used {
			continue
		}

		if strings.HasPrefix(param.Name, arg.Name) && (matchLen < 0 || len(param.Name) < matchLen) {
			match.Idx = idx + numPositional
			matchLen = len(param.Name)
		}
	}
	if match.Idx > -1 {
		return match
	}

	if function.LanguageDetails.Python.Kwarg != nil {
		match.InKwargs = true
		for idx, param := range function.LanguageDetails.Python.KwargParameters {
			if strings.HasPrefix(param.Name, arg.Name) && (matchLen < 0 || len(param.Name) < matchLen) {
				match.Idx = idx
				matchLen = len(param.Name)
			}
		}
	}

	// if there was no match, we end up returning parameter{Idx: -1, InKwargs: true/false} by default

	return match
}

func findParam(function *editorapi.FunctionDetails, arg argument) parameter {
	if arg.InKwarg {
		return findByName(function, arg)
	}
	// otherwise, the user has not yet typed an "="

	// assume function.Parameters has all non-keyword-only params, followed by all keyword-only params
	if arg.Idx < len(function.Parameters) {
		pyDetails := function.Parameters[arg.Idx].LanguageDetails.Python
		if pyDetails == nil || !pyDetails.KeywordOnly {
			// assume the user is typing a positional parameter, since it's possible
			// we could improve this using machine learning or some heuristic approach, based on e.g. names in scope vs keyword names
			return parameter{
				Idx:      arg.Idx,
				InKwargs: false,
			}
		}
	}

	// otherwise, if there are *varargs, that takes precedence over keyword-only params
	if function.LanguageDetails.Python.Vararg != nil {
		return parameter{
			Idx:      len(function.Parameters),
			InKwargs: false,
		}
	}

	// otherwise, we must be starting to type a keyword argument name
	return findByName(function, arg)
}

func parse(src string, start, cursor int64) (arguments, error) {
	defer parseDuration.DeferRecord(time.Now())

	// this should never happen, but just to be safe,
	// since a panic here breaks signatures completely.
	switch {
	case start > cursor:
		invalidParseDelimiters.Add(1)
		invalidParseDelimitersBreakdown.HitAndAdd("start > cursor")
		return arguments{}, fmt.Errorf("invalid parse delimiters: start > cursor")
	case start > int64(len(src)):
		invalidParseDelimiters.Add(1)
		invalidParseDelimitersBreakdown.HitAndAdd("start > len(src)")
		return arguments{}, fmt.Errorf("invalid parse delimiters: start > len(src)")
	case cursor > int64(len(src)):
		invalidParseDelimiters.Add(1)
		invalidParseDelimitersBreakdown.HitAndAdd("cursor > len(src)")
		return arguments{}, fmt.Errorf("invalid parse delimiters: cursor > len(src)")
	}

	// parse the arguments
	args, err := calls.ParseArguments([]byte(src[start:cursor]))
	if err != nil {
		parseErrs.HitAndAdd(errors.ErrorReason(err).String())
		parseErrRatio.Hit()

		// in certain cases we can get a partial result along with an error
		if args != nil {
			return arguments{
				Args:   args.Args,
				Vararg: args.Vararg,
				Kwarg:  args.Kwarg,
				Commas: args.Commas,
			}, nil
		}

		return arguments{}, fmt.Errorf("unable to parse arguments: %v", err)
	}
	parseErrRatio.Miss()

	return arguments{
		Args:   args.Args,
		Vararg: args.Vararg,
		Kwarg:  args.Kwarg,
		Commas: args.Commas,
	}, nil
}

// Finds the start of the arguments for the current call
// or returns -1. The start of the arguments
// is defined to be the position of the openining
// parenthesis for the call.
// The cursor is zero based, and denotes the position of
// the cursor in the file the user is editing.
// The character in the buffer at the position of the cursor
// is always the character immediately in front of the cursor.
// e.g consider `$foo()` the cursor is at position
// zero and the character at position 0 is f.
// NOTE: this means that when the cursor is at the
// end of the file there is no corresponding character.
func findArgsStart(src string, cursor int64) int64 {
	if cursor > int64(len(src)) {
		// should not happen but just to be safe
		cursor = int64(len(src))
		invalidCursorCount.Add(1)
	}

	var paren int64
	var seen bool
	idx := cursor - 1
	for ; idx > -1; idx-- {
		switch src[idx] {
		case '(':
			if paren == 0 {
				failFindArgsStartRatio.Miss()
				return idx
			}
			seen = true
			paren--
		case ')':
			paren++
		default:
		}
	}

	if !seen || idx < 0 {
		failFindArgsStartRatio.Hit()
		return -1
	}

	failFindArgsStartRatio.Miss()
	return idx
}
