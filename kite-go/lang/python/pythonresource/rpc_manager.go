package pythonresource

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/docs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const serviceName = "PythonResourceManager"

var errNilResponse = errors.New("<nil>")

// StartServerDefaultOpts starts a new net/rpc based pythonresource.Manager server, which listens to the given interface and port
func StartServerDefaultOpts(interfaceAndPort string, logging bool) (io.Closer, net.Addr, error) {
	return StartServer(interfaceAndPort, logging, DefaultOptions)
}

// StartServer starts a new net/rpc based pythonresource.Manager server, which listens to the given interface and port
func StartServer(interfaceAndPort string, logging bool, options Options) (io.Closer, net.Addr, error) {
	err := datadeps.Enable()
	if err != nil {
		return nil, nil, err
	}

	datadeps.SetLocalOnly()

	mgr, errChan := NewManager(options)
	if err := <-errChan; err != nil {
		return nil, nil, err
	}

	if logging {
		mgr = NewStatsManager(mgr, false)
	}

	remoteMgr := &rpcManager{
		mgr:     mgr,
		logging: logging,
	}

	rpc.RegisterName(serviceName, remoteMgr)
	l, e := net.Listen("tcp", interfaceAndPort)
	if e != nil {
		log.Fatal("listen error:", e)
	}

	go rpc.DefaultServer.Accept(l)

	remoteMgr.listener = l
	return l, l.Addr(), nil
}

// rpcManager wraps a pythonresource.Manager to make it suitable to be used with net/rpc
type rpcManager struct {
	mgr      Manager
	logging  bool
	listener net.Listener
}

// Close implements io.Closer
func (r *rpcManager) Close() error {
	_ = r.mgr.Close()
	return r.listener.Close()
}

func (r *rpcManager) log(msg ...interface{}) {
	if r.logging {
		fmt.Println(msg...)
	}
}

func (r *rpcManager) Reset(ignored struct{}, ignoredResponse *bool) error {
	//r.mgr.Reset()
	*ignoredResponse = true
	return nil
}

func (r *rpcManager) Distributions(ignored struct{}, response *[]keytypes.Distribution) error {
	*response = r.mgr.Distributions()
	return nil
}

func (r *rpcManager) DistLoaded(request keytypes.Distribution, response *bool) error {
	*response = r.mgr.DistLoaded(request)
	return nil
}

func (r *rpcManager) ArgSpec(request Symbol, response *pythonimports.ArgSpec) error {
	result := r.mgr.ArgSpec(request)
	if result == nil {
		return errNilResponse
	}
	*response = *result
	return nil
}

func (r *rpcManager) PopularSignatures(request Symbol, response *[]*editorapi.Signature) error {
	*response = r.mgr.PopularSignatures(request)
	return nil
}

func (r *rpcManager) CumulativeNumArgsFrequency(request NumArgsFrequencyRequest, response *FloatBoolResponse) error {
	f, b := r.mgr.CumulativeNumArgsFrequency(request.Symbol, request.NumArgs)
	*response = FloatBoolResponse{Float: f, Bool: b}
	return nil
}

func (r *rpcManager) KeywordArgFrequency(request KeywordArgFrequencyRequest, response *IntBoolResponse) error {
	i, b := r.mgr.KeywordArgFrequency(request.Symbol, request.Arg)
	*response = IntBoolResponse{Int: i, Bool: b}
	return nil
}

func (r *rpcManager) NumArgsFrequency(request NumArgsFrequencyRequest, response *FloatBoolResponse) error {
	f, b := r.mgr.NumArgsFrequency(request.Symbol, request.NumArgs)
	*response = FloatBoolResponse{Float: f, Bool: b}
	return nil
}

func (r *rpcManager) Documentation(request Symbol, response *docs.Entity) error {
	documentation := r.mgr.Documentation(request)
	if documentation == nil {
		return errNilResponse
	}
	*response = *documentation
	return nil
}

func (r *rpcManager) SymbolCounts(request Symbol, response *symbolcounts.Counts) error {
	counts := r.mgr.SymbolCounts(request)
	if counts == nil {
		return errNilResponse
	}

	*response = *counts
	return nil
}

func (r *rpcManager) Kwargs(request Symbol, response *KeywordArgs) error {
	kwargs := r.mgr.Kwargs(request)
	if kwargs == nil {
		return errNilResponse
	}
	*response = *kwargs
	return nil
}

func (r *rpcManager) TruthyReturnTypes(request Symbol, response *[]TruthySymbol) error {
	types := r.mgr.TruthyReturnTypes(request)
	if types == nil {
		return errNilResponse
	}
	*response = types
	return nil
}

func (r *rpcManager) ReturnTypes(request Symbol, response *[]Symbol) error {
	types := r.mgr.ReturnTypes(request)
	if types == nil {
		return errNilResponse
	}
	*response = types
	return nil
}

func (r *rpcManager) PathSymbol(request pythonimports.DottedPath, response *Symbol) error {
	sym, err := r.mgr.PathSymbol(request)
	if err != nil {
		return err
	}
	*response = sym
	return nil
}

func (r *rpcManager) PathSymbols(request pythonimports.DottedPath, response *[]Symbol) error {
	// hack, but net/rpc doesn't seem to provide a context anywhere
	symbols, err := r.mgr.PathSymbols(kitectx.Background(), request)
	if err != nil {
		return err
	}
	*response = symbols
	return nil
}

func (r *rpcManager) NewSymbol(request NewSymbolRequest, response *Symbol) error {
	sym, err := r.mgr.NewSymbol(request.Dist, request.Path)
	if err != nil {
		return err
	}
	*response = sym
	return nil
}

func (r *rpcManager) Kind(request Symbol, response *keytypes.Kind) error {
	*response = r.mgr.Kind(request)
	return nil
}

func (r *rpcManager) Type(request Symbol, response *Symbol) error {
	symbol, err := r.mgr.Type(request)
	if err != nil {
		return err
	}
	*response = symbol
	return nil
}

func (r *rpcManager) Bases(request Symbol, response *[]Symbol) error {
	*response = r.mgr.Bases(request)
	return nil
}

func (r *rpcManager) Children(request Symbol, response *[]string) error {
	resp, err := r.mgr.Children(request)
	if err != nil {
		return err
	}
	*response = resp
	return nil
}

func (r *rpcManager) ChildSymbol(request ChildSymbolRequest, response *Symbol) error {
	resp, err := r.mgr.ChildSymbol(request.Symbol, request.String)
	if err != nil {
		return err
	}
	*response = resp
	return nil
}

func (r *rpcManager) CanonicalSymbols(request keytypes.Distribution, response *[]Symbol) error {
	resp, err := r.mgr.CanonicalSymbols(request)
	if err != nil {
		return err
	}
	*response = resp
	return nil
}

func (r *rpcManager) TopLevels(request keytypes.Distribution, response *[]string) error {
	resp, err := r.mgr.TopLevels(request)
	if err != nil {
		return err
	}
	*response = resp
	return nil
}

func (r *rpcManager) Pkgs(ignored struct{}, response *[]string) error {
	*response = r.mgr.Pkgs()
	return nil
}

func (r *rpcManager) DistsForPkg(request string, response *[]keytypes.Distribution) error {
	*response = r.mgr.DistsForPkg(request)
	return nil
}

func (r *rpcManager) SigStats(request Symbol, response *SigStats) error {
	stats := r.mgr.SigStats(request)
	if stats == nil {
		return errNilResponse
	}
	*response = *stats
	return nil
}
