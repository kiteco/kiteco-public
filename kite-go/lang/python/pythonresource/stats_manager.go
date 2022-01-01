package pythonresource

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
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

type argStats struct {
	argString string
	count     int
}

type argStatsMap map[string]*argStats

func (m argStatsMap) StatsString(o io.Writer, methodName string, totalCallCount int) {
	var allArgs []*argStats
	for _, a := range m {
		allArgs = append(allArgs, a)
	}

	sort.Slice(allArgs, func(i, j int) bool {
		return allArgs[i].count > allArgs[j].count
	})

	tab := tabwriter.NewWriter(o, 3, 4, 2, ' ', tabwriter.AlignRight)
	defer tab.Flush()

	duplicateCalls := 0
	for _, arg := range allArgs {
		if arg.count >= 2 {
			duplicateCalls += arg.count - 1
		}
	}

	fmt.Fprintf(o, "\n\n%s: Duplicate calls: %d of %d, %.2f%% of total number of calls\n", methodName, duplicateCalls, totalCallCount, float64(duplicateCalls)/float64(totalCallCount)*100)

	fmt.Fprintf(tab, "Arg\tCount\t%% of total\t\n")
	threshold := 10
	for _, arg := range allArgs {
		if arg.count > threshold {
			fmt.Fprintf(tab, "%s: %s\t%d\t%.2f%%\t\n", methodName, arg.argString, arg.count, float64(arg.count)/(float64(totalCallCount))*100)
		}
	}
}

type callStats struct {
	name          string
	count         int
	totalDuration time.Duration
	argStats      argStatsMap
}

// NewStatsManager wraps another manager and records statistics about called method calls and arguments
func NewStatsManager(mgr Manager, printStats bool) Manager {
	stats := &statsManagerWrapper{
		mgr:         mgr,
		callDetails: make(map[string]*callStats),
	}

	if printStats {
		go func() {
			time.Sleep(time.Minute)
			stats.printCallStatus()
		}()
	}

	return stats
}

// loggingManager implements loggingManager and logs details on every call
type statsManagerWrapper struct {
	mgr     Manager
	logging bool

	mu          sync.Mutex
	callDetails map[string]*callStats
}

func (m *statsManagerWrapper) logCall(name string, argString string) func() {
	if m.logging {
		log.Printf("pythonresource.Manager: %s()\n", name)
	}

	start := time.Now()
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		details, ok := m.callDetails[name]
		if !ok {
			details = &callStats{
				name:     name,
				argStats: make(argStatsMap),
			}
			m.callDetails[name] = details
		}

		details.count++
		details.totalDuration += time.Now().Sub(start)

		arg, ok := details.argStats[argString]
		if !ok {
			arg = &argStats{
				argString: argString,
			}
			details.argStats[argString] = arg
		}
		arg.count++
	}
}

func (m *statsManagerWrapper) printCallStatus() {
	m.mu.Lock()
	defer m.mu.Unlock()

	//defer func() {
	//	m.callDetails = make(map[string]*callStats)
	//}()

	var all []*callStats
	for _, status := range m.callDetails {
		all = append(all, status)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].totalDuration > all[j].totalDuration
	})

	w := tabwriter.Writer{}
	w.Init(os.Stdout, 6, 4, 2, ' ', tabwriter.AlignRight|tabwriter.TabIndent)

	fmt.Println("Total durations:")
	fmt.Fprintf(&w, "Name\t Calls\t Total duration (s)\t per call (ms)\t\n")
	for _, status := range all {
		fmt.Fprintf(&w, "%s \t %d \t %.3f \t %.3f\t\n", status.name, status.count, status.totalDuration.Seconds(), float64(status.totalDuration.Milliseconds())/float64(status.count))
		w.Flush()

		status.argStats.StatsString(&w, status.name, status.count)
	}
	fmt.Fprintln(&w)
	w.Flush()
}

func (m *statsManagerWrapper) Close() error {
	defer m.logCall("Close", "")()
	return m.mgr.Close()
}

func (m *statsManagerWrapper) Reset() {
	defer m.logCall("Reset", "")()
	m.mgr.Reset()
}

func (m *statsManagerWrapper) Distributions() []keytypes.Distribution {
	defer m.logCall("Distributions", "")()
	return m.mgr.Distributions()
}

func (m *statsManagerWrapper) DistLoaded(dist keytypes.Distribution) bool {
	defer m.logCall("DistLoaded", dist.String())()
	return m.mgr.DistLoaded(dist)
}

func (m *statsManagerWrapper) ArgSpec(sym Symbol) *pythonimports.ArgSpec {
	defer m.logCall("ArgSpec", sym.String())()
	return m.mgr.ArgSpec(sym)
}

func (m *statsManagerWrapper) PopularSignatures(sym Symbol) []*editorapi.Signature {
	defer m.logCall("PopularSignatures", sym.String())()
	return m.mgr.PopularSignatures(sym)
}

func (m *statsManagerWrapper) CumulativeNumArgsFrequency(sym Symbol, numArgs int) (float64, bool) {
	defer m.logCall("CumulativeNumArgsFrequency", fmt.Sprintf("%s_%d", sym.String(), numArgs))()
	return m.mgr.CumulativeNumArgsFrequency(sym, numArgs)
}

func (m *statsManagerWrapper) KeywordArgFrequency(sym Symbol, arg string) (int, bool) {
	defer m.logCall("KeywordArgFrequency", fmt.Sprintf("%s_%s", sym.String(), arg))()
	return m.mgr.KeywordArgFrequency(sym, arg)
}

func (m *statsManagerWrapper) NumArgsFrequency(sym Symbol, numArgs int) (float64, bool) {
	defer m.logCall("NumArgsFrequency", fmt.Sprintf("%s_%d", sym.String(), numArgs))()
	return m.mgr.NumArgsFrequency(sym, numArgs)
}

func (m *statsManagerWrapper) Documentation(sym Symbol) *docs.Entity {
	defer m.logCall("Documentation", sym.String())()
	return m.mgr.Documentation(sym)
}

func (m *statsManagerWrapper) SymbolCounts(sym Symbol) *symbolcounts.Counts {
	defer m.logCall("SymbolCounts", sym.String())()
	return m.mgr.SymbolCounts(sym)
}

func (m *statsManagerWrapper) Kwargs(sym Symbol) *KeywordArgs {
	defer m.logCall("Kwargs", sym.String())()
	return m.mgr.Kwargs(sym)
}

func (m *statsManagerWrapper) TruthyReturnTypes(sym Symbol) []TruthySymbol {
	defer m.logCall("TruthyReturnTypes", sym.String())()
	return m.mgr.TruthyReturnTypes(sym)
}

func (m *statsManagerWrapper) ReturnTypes(sym Symbol) []Symbol {
	defer m.logCall("ReturnTyppes", sym.String())()
	return m.mgr.ReturnTypes(sym)
}

func (m *statsManagerWrapper) PathSymbol(path pythonimports.DottedPath) (Symbol, error) {
	defer m.logCall("PathSymbol", path.String())()
	return m.mgr.PathSymbol(path)
}

func (m *statsManagerWrapper) PathSymbols(ctx kitectx.Context, path pythonimports.DottedPath) ([]Symbol, error) {
	defer m.logCall("PathSymbols", path.String())()
	return m.mgr.PathSymbols(ctx, path)
}

func (m *statsManagerWrapper) NewSymbol(dist keytypes.Distribution, path pythonimports.DottedPath) (Symbol, error) {
	defer m.logCall("NewSymbol", fmt.Sprintf("%s_%s", dist.String(), path.String()))()
	return m.mgr.NewSymbol(dist, path)
}

func (m *statsManagerWrapper) Kind(s Symbol) keytypes.Kind {
	defer m.logCall("Kind", s.String())()
	return m.mgr.Kind(s)
}

func (m *statsManagerWrapper) Type(s Symbol) (Symbol, error) {
	defer m.logCall("Type", s.String())()
	return m.mgr.Type(s)
}

func (m *statsManagerWrapper) Bases(s Symbol) []Symbol {
	defer m.logCall("Bases", s.String())()
	return m.mgr.Bases(s)
}

func (m *statsManagerWrapper) Children(s Symbol) ([]string, error) {
	defer m.logCall("Children", s.String())()
	return m.mgr.Children(s)
}

func (m *statsManagerWrapper) ChildSymbol(s Symbol, c string) (Symbol, error) {
	defer m.logCall("ChildSymbol", fmt.Sprintf("%s_%s", s.String(), c))()
	return m.mgr.ChildSymbol(s, c)
}

func (m *statsManagerWrapper) CanonicalSymbols(dist keytypes.Distribution) ([]Symbol, error) {
	defer m.logCall("CanonicalSymbols", dist.String())()
	return m.mgr.CanonicalSymbols(dist)
}

func (m *statsManagerWrapper) TopLevels(dist keytypes.Distribution) ([]string, error) {
	defer m.logCall("TopLevels", dist.String())()
	return m.mgr.TopLevels(dist)
}

func (m *statsManagerWrapper) Pkgs() []string {
	defer m.logCall("Pkgs", "")()
	return m.mgr.Pkgs()
}

func (m *statsManagerWrapper) DistsForPkg(pkg string) []keytypes.Distribution {
	defer m.logCall("DistsForPkg", pkg)()
	return m.mgr.DistsForPkg(pkg)
}

func (m *statsManagerWrapper) SigStats(sym Symbol) *SigStats {
	defer m.logCall("SigStats", sym.String())()
	return m.mgr.SigStats(sym)
}
