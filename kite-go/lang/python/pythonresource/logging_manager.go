package pythonresource

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/docs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type callDetails struct {
	name          string
	count         int
	totalDuration time.Duration
}

// NewLoggingManager wraps another manager and records simple statistics about the called methods of that manager
func NewLoggingManager(mgr Manager, printStatus bool, loggingPrefix string) Manager {
	logger := &loggingManager{
		mgr:         mgr,
		callDetails: make(map[string]*callDetails),
	}

	if printStatus {
		go func() {
			for {
				time.Sleep(time.Minute)
				logger.printCallStatus(loggingPrefix)
			}
		}()
	}

	return logger
}

// loggingManager implements loggingManager and logs details on every call
type loggingManager struct {
	mgr     Manager
	logging bool

	mu          sync.Mutex
	callDetails map[string]*callDetails
}

func (m *loggingManager) logCall(name string) func() {
	if m.logging {
		log.Printf("pythonresource.Manager: %s()\n", name)
	}

	start := time.Now()
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		details, ok := m.callDetails[name]
		if !ok {
			details = &callDetails{
				name: name,
			}
			m.callDetails[name] = details
		}

		details.count++
		details.totalDuration += time.Now().Sub(start)
	}
}

func (m *loggingManager) printCallStatus(prefix string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var all []*callDetails
	for _, status := range m.callDetails {
		all = append(all, status)
	}
	sort.Slice(all, func(i, j int) bool {
		//return all[i].totalDuation > all[j].totalDuration
		return strings.Compare(all[i].name, all[j].name) < 0
	})

	out := strings.Builder{}
	out.WriteString("===== " + prefix + " =====\n")
	w := tabwriter.Writer{}
	w.Init(&out, 6, 4, 2, ' ', tabwriter.AlignRight|tabwriter.TabIndent)

	fmt.Println("Total durations:")
	fmt.Fprintf(&w, "Name\t Calls\t Total duration (s)\t per call (ms)\t\n")
	for _, status := range all {
		fmt.Fprintf(&w, "%s \t %d \t %.3f \t %.3f\t\n", status.name, status.count, status.totalDuration.Seconds(), float64(status.totalDuration.Milliseconds())/float64(status.count))
	}
	fmt.Fprintln(&w)
	w.Flush()

	println(out.String())
}

func (m *loggingManager) Close() error {
	defer m.logCall("Close")()
	return m.mgr.Close()
}

func (m *loggingManager) Reset() {
	defer m.logCall("Reset")()
	m.mgr.Reset()
}

func (m *loggingManager) Distributions() []keytypes.Distribution {
	defer m.logCall("Distributions")()
	return m.mgr.Distributions()
}

func (m *loggingManager) DistLoaded(dist keytypes.Distribution) bool {
	defer m.logCall("DistLoaded")()
	return m.mgr.DistLoaded(dist)
}

func (m *loggingManager) ArgSpec(sym Symbol) *pythonimports.ArgSpec {
	defer m.logCall("ArgSpec")()
	return m.mgr.ArgSpec(sym)
}

func (m *loggingManager) PopularSignatures(sym Symbol) []*editorapi.Signature {
	defer m.logCall("PopularSignatures")()
	return m.mgr.PopularSignatures(sym)
}

func (m *loggingManager) CumulativeNumArgsFrequency(sym Symbol, numArgs int) (float64, bool) {
	defer m.logCall("CumulativeNumArgsFrequency")()
	return m.mgr.CumulativeNumArgsFrequency(sym, numArgs)
}

func (m *loggingManager) KeywordArgFrequency(sym Symbol, arg string) (int, bool) {
	defer m.logCall("KeywordArgFrequency")()
	return m.mgr.KeywordArgFrequency(sym, arg)
}

func (m *loggingManager) NumArgsFrequency(sym Symbol, numArgs int) (float64, bool) {
	defer m.logCall("NumArgsFrequency")()
	return m.mgr.NumArgsFrequency(sym, numArgs)
}

func (m *loggingManager) Documentation(sym Symbol) *docs.Entity {
	defer m.logCall("Documentation")()
	return m.mgr.Documentation(sym)
}

func (m *loggingManager) SymbolCounts(sym Symbol) *symbolcounts.Counts {
	defer m.logCall("SymbolCounts")()
	return m.mgr.SymbolCounts(sym)
}

func (m *loggingManager) Kwargs(sym Symbol) *KeywordArgs {
	defer m.logCall("Kwargs")()
	return m.mgr.Kwargs(sym)
}

func (m *loggingManager) TruthyReturnTypes(sym Symbol) []TruthySymbol {
	defer m.logCall("TruthyReturnTypes")()
	return m.mgr.TruthyReturnTypes(sym)
}

func (m *loggingManager) ReturnTypes(sym Symbol) []Symbol {
	defer m.logCall("ReturnTyppes")()
	return m.mgr.ReturnTypes(sym)
}

func (m *loggingManager) PathSymbol(path pythonimports.DottedPath) (Symbol, error) {
	defer m.logCall("PathSymbol")()
	return m.mgr.PathSymbol(path)
}

func (m *loggingManager) PathSymbols(ctx kitectx.Context, path pythonimports.DottedPath) ([]Symbol, error) {
	defer m.logCall("PathSymbols")()
	return m.mgr.PathSymbols(ctx, path)
}

func (m *loggingManager) NewSymbol(dist keytypes.Distribution, path pythonimports.DottedPath) (Symbol, error) {
	defer m.logCall("NewSymbol")()
	return m.mgr.NewSymbol(dist, path)
}

func (m *loggingManager) Kind(s Symbol) keytypes.Kind {
	defer m.logCall("Kind")()
	return m.mgr.Kind(s)
}

func (m *loggingManager) Type(s Symbol) (Symbol, error) {
	defer m.logCall("Type")()
	return m.mgr.Type(s)
}

func (m *loggingManager) Bases(s Symbol) []Symbol {
	defer m.logCall("Bases")()
	return m.mgr.Bases(s)
}

func (m *loggingManager) Children(s Symbol) ([]string, error) {
	defer m.logCall("Children")()
	return m.mgr.Children(s)
}

func (m *loggingManager) ChildSymbol(s Symbol, c string) (Symbol, error) {
	defer m.logCall("ChildSymbol")()
	return m.mgr.ChildSymbol(s, c)
}

func (m *loggingManager) CanonicalSymbols(dist keytypes.Distribution) ([]Symbol, error) {
	defer m.logCall("CanonicalSymbols")()
	return m.mgr.CanonicalSymbols(dist)
}

func (m *loggingManager) TopLevels(dist keytypes.Distribution) ([]string, error) {
	defer m.logCall("TopLevels")()
	return m.mgr.TopLevels(dist)
}

func (m *loggingManager) Pkgs() []string {
	defer m.logCall("Pkgs")()
	return m.mgr.Pkgs()
}

func (m *loggingManager) DistsForPkg(pkg string) []keytypes.Distribution {
	defer m.logCall("DistsForPkg")()
	return m.mgr.DistsForPkg(pkg)
}

func (m *loggingManager) SigStats(sym Symbol) *SigStats {
	defer m.logCall("SigStats")()
	return m.mgr.SigStats(sym)
}
