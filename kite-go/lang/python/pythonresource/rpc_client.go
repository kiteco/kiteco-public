package pythonresource

import (
	"fmt"
	"net/rpc"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/docs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// NewRPCClient returns a pythonresource.Manager, which talks to a remote server at the given host and port
func NewRPCClient(serverAndPort string) (Manager, error) {
	client, err := rpc.Dial("tcp", serverAndPort)
	if err != nil {
		return nil, err
	}

	return &rpcClientManager{client: client}, nil
}

var noArgs = struct{}{}

// rpcClientManager implements pythonresource.Manager, which talks to a remote server
type rpcClientManager struct {
	client     *rpc.Client
	errorCount int
}

func (r *rpcClientManager) testReset() {
	r.errorCount = 0
}

// simpleCall is for remote calls, which take a single parameter and don't return an error
// it returns true if a result is available, false indicates a response of nil
func (r *rpcClientManager) simpleCall(name string, param interface{}, response interface{}) bool {
	err := r.client.Call(fmt.Sprintf("%s.%s", serviceName, name), param, response)
	if err != nil && err.Error() != "<nil>" {
		r.errorCount++
	}
	return err == nil
}

// errorCall is for remote calls, which take a single parameter and returns an error
func (r *rpcClientManager) errorCall(name string, param interface{}, response interface{}) error {
	err := r.client.Call(fmt.Sprintf("%s.%s", serviceName, name), param, response)
	if err != nil && err.Error() == "<nil>" {
		return errNilResponse
	}

	if err != nil {
		r.errorCount++
	}
	return err
}

func (r *rpcClientManager) Close() error {
	return r.client.Close()
}

func (r *rpcClientManager) Reset() {
	r.simpleCall("Reset", noArgs, nil)
}

func (r *rpcClientManager) Distributions() []keytypes.Distribution {
	var dists []keytypes.Distribution
	r.simpleCall("Distributions", noArgs, &dists)
	return dists
}

func (r *rpcClientManager) DistLoaded(dist keytypes.Distribution) bool {
	var loaded bool
	r.simpleCall("DistLoaded", dist, &loaded)
	return loaded
}

func (r *rpcClientManager) ArgSpec(sym Symbol) *pythonimports.ArgSpec {
	var response pythonimports.ArgSpec
	if !r.simpleCall("ArgSpec", sym, &response) {
		return nil
	}
	return &response
}

func (r *rpcClientManager) PopularSignatures(sym Symbol) []*editorapi.Signature {
	var response []*editorapi.Signature
	r.simpleCall("PopularSignatures", sym, &response)
	return response
}

func (r *rpcClientManager) CumulativeNumArgsFrequency(sym Symbol, numArgs int) (float64, bool) {
	request := NumArgsFrequencyRequest{Symbol: sym, NumArgs: numArgs}

	var response FloatBoolResponse
	r.simpleCall("CumulativeNumArgsFrequency", request, &response)
	return response.Float, response.Bool
}

func (r *rpcClientManager) KeywordArgFrequency(sym Symbol, arg string) (int, bool) {
	request := KeywordArgFrequencyRequest{Symbol: sym, Arg: arg}

	var response IntBoolResponse
	r.simpleCall("KeywordArgFrequency", request, &response)
	return response.Int, response.Bool
}

func (r *rpcClientManager) NumArgsFrequency(sym Symbol, numArgs int) (float64, bool) {
	request := NumArgsFrequencyRequest{Symbol: sym, NumArgs: numArgs}

	var response FloatBoolResponse
	r.simpleCall("NumArgsFrequency", request, &response)
	return response.Float, response.Bool
}

func (r *rpcClientManager) Documentation(sym Symbol) *docs.Entity {
	var response docs.Entity
	if !r.simpleCall("Documentation", sym, &response) {
		return nil
	}
	return &response
}

func (r *rpcClientManager) SymbolCounts(sym Symbol) *symbolcounts.Counts {
	var response symbolcounts.Counts
	if !r.simpleCall("SymbolCounts", sym, &response) {
		return nil
	}
	return &response
}

func (r *rpcClientManager) Kwargs(sym Symbol) *KeywordArgs {
	var response KeywordArgs
	if !r.simpleCall("Kwargs", sym, &response) {
		return nil
	}
	return &response
}

func (r *rpcClientManager) TruthyReturnTypes(sym Symbol) []TruthySymbol {
	var response []TruthySymbol
	r.simpleCall("TruthyReturnTypes", sym, &response)
	return response
}

func (r *rpcClientManager) ReturnTypes(sym Symbol) []Symbol {
	var response []Symbol
	r.simpleCall("ReturnTypes", sym, &response)
	return response
}

func (r *rpcClientManager) PathSymbol(path pythonimports.DottedPath) (Symbol, error) {
	var response Symbol
	err := r.errorCall("PathSymbol", path, &response)
	return response, err
}

func (r *rpcClientManager) PathSymbols(ctx kitectx.Context, path pythonimports.DottedPath) ([]Symbol, error) {
	ctx.CheckAbort()

	var response []Symbol
	err := r.errorCall("PathSymbols", path, &response)
	return response, err
}

func (r *rpcClientManager) NewSymbol(dist keytypes.Distribution, path pythonimports.DottedPath) (Symbol, error) {
	var response Symbol
	err := r.errorCall("NewSymbol", NewSymbolRequest{Dist: dist, Path: path}, &response)
	return response, err
}

func (r *rpcClientManager) Kind(s Symbol) keytypes.Kind {
	var response keytypes.Kind
	r.simpleCall("Kind", s, &response)
	return response
}

func (r *rpcClientManager) Type(s Symbol) (Symbol, error) {
	var response Symbol
	err := r.errorCall("Type", s, &response)
	return response, err
}

func (r *rpcClientManager) Bases(s Symbol) []Symbol {
	var response []Symbol
	r.simpleCall("Bases", s, &response)
	return response
}

func (r *rpcClientManager) Children(s Symbol) ([]string, error) {
	var response []string
	err := r.errorCall("Children", s, &response)
	return response, err
}

func (r *rpcClientManager) ChildSymbol(s Symbol, c string) (Symbol, error) {
	var response Symbol
	err := r.errorCall("ChildSymbol", ChildSymbolRequest{Symbol: s, String: c}, &response)
	return response, err
}

func (r *rpcClientManager) CanonicalSymbols(dist keytypes.Distribution) ([]Symbol, error) {
	var response []Symbol
	err := r.errorCall("CanonicalSymbols", dist, &response)
	return response, err
}

func (r *rpcClientManager) TopLevels(dist keytypes.Distribution) ([]string, error) {
	var response []string
	err := r.errorCall("TopLevels", dist, &response)
	return response, err
}

func (r *rpcClientManager) Pkgs() []string {
	var response []string
	r.simpleCall("Pkgs", noArgs, &response)
	return response
}

func (r *rpcClientManager) DistsForPkg(pkg string) []keytypes.Distribution {
	var response []keytypes.Distribution
	r.simpleCall("DistsForPkg", pkg, &response)
	return response
}

func (r *rpcClientManager) SigStats(sym Symbol) *SigStats {
	var response SigStats
	if !r.simpleCall("SigStats", sym, &response) {
		return nil
	}
	return &response
}
