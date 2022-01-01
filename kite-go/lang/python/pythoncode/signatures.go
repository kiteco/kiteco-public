package pythoncode

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// SignatureOptions defines the options for SignaturePatterns.
type SignatureOptions struct {
	// Coverage defines the coverage percentage of the signature pattern search. This is used
	// with MaxSignatures - first threshold to trigger wins.
	Coverage float64
	// MinUsage defines the minimum usage percentage for a signature pattern for
	// it to be included in the results.
	MinUsage float64
	// MaxSignatures defines the maximum number of patterns to return. This is used with
	// Coverage - first threshold to trigger wins.
	MaxSignatures int
}

var (
	// DefaultSignatureOptions defines sensible default options for SignaturePatterns.
	DefaultSignatureOptions = SignatureOptions{
		Coverage:      0.9,
		MinUsage:      0.01,
		MaxSignatures: 5,
	}
)

const (
	maxTypes         = 2    // Maximum number of types allowed per argument
	minTypeFrequency = 0.10 // Minimum frequency for a type to be included
)

var (
	exampleSanitizer = regexp.MustCompile("\n\\s*")
)

// SignatureResult contains information about common invokation signatures
type SignatureResult struct {
	Pattern  *SignaturePattern
	Snippets []*Snippet
}

// --

// SignaturePatterns serves common invocation patterns along with examples of
// each pattern returned.
type SignaturePatterns struct {
	signatureIndex map[int64]*MethodPatterns
	graph          *pythonimports.Graph
	opts           SignatureOptions
}

// NewSignaturePatterns creates a new SignaturePattern index.
func NewSignaturePatterns(patterns string, graph *pythonimports.Graph, opts SignatureOptions) (*SignaturePatterns, error) {
	sp := &SignaturePatterns{
		signatureIndex: make(map[int64]*MethodPatterns),
		graph:          graph,
		opts:           opts,
	}
	err := sp.loadPatterns(patterns)
	if err != nil {
		return nil, err
	}

	// in the emr pipeline we group all the usages of a types constructor under the
	// __init__ method for the type (if it exists). This is to make sure that
	// we combine the stats for both forms of constructing an instance of
	// a type while not duplicating the data.
	// TODO(juan): should we just duplicate the data?
	for _, node := range graph.Nodes {
		if node.Classification != pythonimports.Type || sp.signatureIndex[node.ID] != nil {
			continue
		}

		if member := node.Members["__init__"]; member != nil {
			if mp := sp.signatureIndex[member.ID]; mp != nil {
				sp.signatureIndex[node.ID] = mp
			}
		}
	}

	return sp, nil
}

// Index returns the index the underlying index for the patterns.
func (s *SignaturePatterns) Index() map[int64]*MethodPatterns {
	return s.signatureIndex
}

// Find returns SignaturePatterns for the provided identifier.
func (s *SignaturePatterns) Find(ident string) ([]*SignatureResult, bool) {
	if patterns, exists := s.Completions(ident); exists {
		var cdf float64
		var results []*SignatureResult
		for _, pattern := range patterns.Patterns {
			if cdf > s.opts.Coverage || len(results) >= s.opts.MaxSignatures {
				break
			}
			pattern.Signature = pattern.legacySignature(ident)
			results = append(results, &SignatureResult{
				Pattern: pattern,
			})
			cdf += pattern.Frequency
		}
		return results, len(results) > 0
	}
	return nil, false
}

// Completions returns Signature Pattern completions for the provided identifier.
func (s *SignaturePatterns) Completions(ident string) (*MethodPatterns, bool) {
	node, err := s.graph.Find(ident)
	if err != nil || node == nil {
		return nil, false
	}
	return s.CompletionsForNode(node)
}

// CompletionsForNode returns signature patterns for the provided node
func (s *SignaturePatterns) CompletionsForNode(node *pythonimports.Node) (*MethodPatterns, bool) {
	if allPatterns, exists := s.signatureIndex[node.ID]; exists {
		patterns := &MethodPatterns{
			Method:      allPatterns.Method,
			MethodCount: allPatterns.MethodCount,
			Args:        allPatterns.Args,
			Kwargs:      allPatterns.Kwargs,
		}
		for _, pattern := range allPatterns.Patterns {
			if pattern.Frequency < s.opts.MinUsage || len(patterns.Patterns) >= s.opts.MaxSignatures {
				break
			}
			patterns.Patterns = append(patterns.Patterns, pattern)
		}
		return patterns, true
	}
	return nil, false
}

// --

func (s *SignaturePatterns) loadPatterns(path string) error {
	f, err := fileutil.NewCachedReader(path)
	if err != nil {
		return fmt.Errorf("cannot open signature patterns from %s: %v", path, err)
	}
	defer f.Close()

	iter := awsutil.NewEMRIterator(f)
	for iter.Next() {
		var patterns MethodPatterns
		err := json.Unmarshal(iter.Value(), &patterns)
		if err != nil {
			log.Fatalln(err)
		}

		// Find this method in the graph
		node, err := s.graph.Find(patterns.Method)
		if err != nil {
			continue
		}

		ProcessPatterns(&patterns)
		s.signatureIndex[node.ID] = &patterns
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("error reading signature patterns from %s: %v", path, err)
	}
	return nil
}

// ProcessPatterns populates the fields of a MethodPatterns object so that it
// can be used to create signature completions. It is idempotent so it is safe
// to call it multiple times. ProcessPatterns must be called in order for
// signature completions to work correctly.
func ProcessPatterns(mp *MethodPatterns) {
	if mp.processed {
		return
	}

	// Populate args and kwargs for each signature pattern
	for _, pat := range mp.Patterns {
		// Set positional arguments and keyword arguments
		pat.kwargs = make(map[string]*ArgStats)
		for i := 0; i < pat.Args; i++ {
			pat.args = append(pat.args, mp.Args[i])
			pat.all = append(pat.all, mp.Args[i])
		}
		for _, kwarg := range pat.Kwargs {
			pat.kwargs[kwarg] = mp.Kwargs[kwarg]
			pat.all = append(pat.all, mp.Kwargs[kwarg])
		}

		// Sort keyword args by count
		byCount := kwargsByCount{
			kwargs:   pat.Kwargs,
			patterns: mp,
		}
		sort.Sort(sort.Reverse(byCount))
	}

	// Set positional argument types and names
	for _, arg := range mp.Args {
		if arg.Type == "" {
			types := selectTypes(arg.Types)
			arg.Type = strings.Join(types, ",")
		}
		if arg.Name == "" {
			for _, name := range arg.ExprStrs {
				if name.Value != "self" && name.Value != "None" {
					arg.Name = name.Value
					break
				}
			}
		}
	}

	// Set keyword argument types
	for _, arg := range mp.Kwargs {
		// NoneType is usually a placeholder. Check to see if our stats indicate
		// it has a common type other than NoneType.
		if arg.Type == "" || arg.Type == "builtins.None.__class__" {
			types := selectTypes(arg.Types)
			arg.Type = strings.Join(types, ",")
		}
	}

	mp.processed = true
}

// --

// SignatureResponse takes a slice of *SignatureResults and builds a PythonSignaturePatterns
// response object.
func SignatureResponse(results []*SignatureResult) *response.PythonSignaturePatterns {
	var signatures []*response.PythonSignaturePattern
	for _, sig := range results {
		resp := &response.PythonSignaturePattern{
			Signature: sig.Pattern.Signature,
			Frequency: sig.Pattern.Frequency,
		}
		for _, snippet := range sig.Snippets {
			resp.Examples = append(resp.Examples, &response.PythonExample{
				Code:   snippet.Code,
				Source: snippet.From(),
			})
		}
		signatures = append(signatures, resp)
	}
	return &response.PythonSignaturePatterns{
		Type:       response.PythonSignaturePatternType,
		Signatures: signatures,
	}
}

var (
	commaToken = &response.PythonSignatureToken{
		TokenType: "text",
		Token:     ",",
	}
	endToken = &response.PythonSignatureToken{
		TokenType: "text",
		Token:     ")",
	}
)

// UserSignature groups together all information about the user's invocation based on the parsed code.
// This gets used to construct signature completions.
type UserSignature struct {
	FullPrefix string
	Method     string
	ArgIndex   int
	AllArgs    []UserArg
}

// UserArg represents an argument entered by the user.
type UserArg struct {
	Name    string
	Value   string
	Keyword bool
}

// --

// EditorSignatures constructs a signature completions response for use with
// in-editor integrations.
func EditorSignatures(patterns *MethodPatterns) []*editorapi.Signature {
	var sigs []*editorapi.Signature
	for _, pat := range patterns.Patterns {
		sigs = append(sigs, editorSignature(pat))
	}
	return sigs
}

func editorSignature(pattern *SignaturePattern) *editorapi.Signature {
	sig := &editorapi.Signature{}
	for _, arg := range pattern.args {
		sig.Args = append(sig.Args, editorParameterExample(arg))
	}

	seen := make(map[string]bool)

	var kwargs []*editorapi.ParameterExample
	for _, name := range pattern.Kwargs {
		if seen[name] {
			continue
		}
		arg, ok := pattern.kwargs[name]
		if !ok {
			continue
		}
		kwargs = append(kwargs, editorParameterExample(arg))
		seen[name] = true
	}

	sig.LanguageDetails.Python = &editorapi.PythonSignatureDetails{
		Kwargs: kwargs,
	}

	return sig
}

func editorParameterExample(arg *ArgStats) *editorapi.ParameterExample {
	var types []*editorapi.ParameterTypeExample
	argType := arg.Type
	for i, t := range splitTypes(argType) {
		if i > 3 {
			break
		}
		types = append(types, &editorapi.ParameterTypeExample{
			ID:       editorapi.NewID(lang.Python, t),
			Name:     t[strings.LastIndex(t, ".")+1:],
			Examples: topLiteralsN(t, arg.LiteralsByType[t], 3),
		})
	}
	if argType == "" || len(arg.ExprStrs) > 0 {
		types = append(types, &editorapi.ParameterTypeExample{
			Name:     "unknown",
			Examples: topLiteralsN("", arg.ExprStrs, 3),
		})
	}
	return &editorapi.ParameterExample{
		Name:  arg.Name,
		Types: types,
	}
}

func patternPrefix(pattern *SignaturePattern, typed *UserSignature) (bool, []string) {
	// If what we typed has more arguments than the pattern, ignore
	if len(typed.AllArgs) > len(pattern.all) {
		return false, nil
	}

	var matchedList []string
	matched := make(map[string]bool)
	for i := 0; i < len(typed.AllArgs) && i < len(pattern.all); i++ {
		patternArg := pattern.all[i]
		userArg := typed.AllArgs[i]
		typedArg := userArg.Name
		if typedArg == "" {
			// for the recursive descent parser Name will be an
			// empty string if it is not a keyword argument
			typedArg = userArg.Value
		}

		// If the pattern contains a kwarg in this position:
		// - Check to see if the typed kwarg matches one in the pattern
		// - If not, check to see if the typed kwarg prefix matches any kwarg in the pattern
		// - Keep track of matched kwargs so we don't allow matching any kwarg twice
		if patternArg.Keyword {
			// If we are not on the last typed argument, ensure that
			// what the user has typed is also actually a keyword.
			if i != len(typed.AllArgs)-1 && !userArg.Keyword {
				return false, nil
			}
			if _, exists := pattern.kwargs[typedArg]; exists {
				matched[typedArg] = true
				matchedList = append(matchedList, typedArg)
			} else {
				var hasPrefix bool
				for pkwarg := range pattern.kwargs {
					if _, exists := matched[pkwarg]; exists {
						continue
					}
					if strings.HasPrefix(pkwarg, typedArg) {
						matched[typedArg] = true
						matchedList = append(matchedList, pkwarg)
						hasPrefix = true
						break
					}
				}
				if !hasPrefix {
					return false, nil
				}
			}
		} else if userArg.Keyword {
			// If the pattern argument was not a keyword but the user has typed a
			// keyword, this pattern is no longer valid.
			return false, nil
		}
	}
	return true, matchedList
}

func topLiteralsN(argType string, values []*StringCount, n int) []string {
	if len(values) > n {
		values = values[:n]
	}
	var ret []string
	for _, val := range values {
		if argType == "builtins.str" {
			ret = append(ret, string(exampleSanitizer.ReplaceAll([]byte(fmt.Sprintf("\"%s\"", val.Value)), []byte(""))))
		} else {
			ret = append(ret, string(exampleSanitizer.ReplaceAll([]byte(val.Value), []byte(""))))
		}
	}
	return ret
}

func splitTypes(varType string) []string {
	return strings.Split(varType, ",")
}

// Select types that make up > minTypeFrequency of all types observed
func selectTypes(argTypes []*StringCount) []string {
	var total int
	for _, varType := range argTypes {
		if varType.Value != "unknown" {
			total += varType.Count
		}
	}
	var types []string
	for _, varType := range argTypes {
		if float64(varType.Count)/float64(total) > minTypeFrequency {
			// TODO(tarak): Bit of a hack to only show fixed number of types on UI
			if len(types) >= maxTypes {
				types = append(types, "...")
				break
			}
			types = append(types, varType.Value)
		}
	}
	return types
}
