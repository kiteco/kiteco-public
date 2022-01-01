package pythonstatic

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type edgeStr struct {
	S, E string
}

func assertCapabilities(t *testing.T, expected []string, actual []Capability) {
	expectedMap := make(map[string]int)
	for _, ec := range expected {
		expectedMap[ec]++
	}

	actualMap := make(map[string]int)
	for _, ac := range actual {
		actualMap[ac.Attr]++
	}

	for a, c := range expectedMap {
		if actualMap[a] != c {
			t.Errorf("  for attr %s: expected %d, actual %d\n", a, c, actualMap[a])
		}
	}

	for a, c := range actualMap {
		if _, found := expectedMap[a]; !found {
			t.Errorf("  got extra Capability %s with count %d\n", a, c)
		}
	}
}

func assertTypesAndCapabilities(
	t *testing.T,
	src string,
	rm pythonresource.Manager,
	expectedValues map[string]pythontype.Value,
	expectedEdges []edgeStr,
	expectedCapabilities map[string][]string,
) {
	opts := DefaultOptions
	opts.UseCapabilities = true
	ai := AssemblerInputs{
		Graph: rm,
	}
	assembler := NewAssembler(kitectx.Background(), ai, opts)

	srcs := map[string]string{
		"/code/src.py": src,
	}

	actual := assertAssemblerBatch(t, srcs, assembler, expectedValues)

	// check edges
	for _, expected := range expectedEdges {
		var found bool
		for s, neighbors := range assembler.helpers.CapabilityDelegate.ForwardGraph() {
			as := s.Name.Path.String()
			for _, neighbor := range neighbors {
				ae := neighbor.Name.Path.String()
				if as == expected.S && ae == expected.E {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Errorf("unable to find expected edge %s->%s\n", expected.S, expected.E)
		}
	}

	// check capabilities for each symbol
	for key, caps := range expectedCapabilities {
		sym, found := actual[key]
		if !found {
			t.Errorf("no symbol %s found\n", key)
			continue
		}

		t.Logf("veryfying capabilities for %s\n", sym.Name.Path.String())
		assertCapabilities(t, caps, assembler.helpers.CapabilityDelegate.capabilities[sym])
	}
}

func TestCapabilities(t *testing.T) {
	src := `
class Server():
    def __init__(self):
        self.port = None
        self.name = None
        self.running = False

    def start(self):
        self.running = true

    def isRunning(self):
        return self.running

def startServer(s, port):
    s.port = port
    s.start()

def foo():
	return True

def main(s):
    startServer(s, ":9091")
    running = foo()
`
	expectedEdges := []edgeStr{
		{S: "main.s", E: "startServer.s"},
		{S: "foo.[return]", E: "main.running"},
	}

	expectedCaps := map[string][]string{
		"startServer.s":         []string{"port", "start"},
		"Server.start.self":     []string{"running"},
		"Server.isRunning.self": []string{"running"},
		"Server.__init__.self":  []string{"port", "name", "running"},
	}

	assertTypesAndCapabilities(t, src, pythonresource.MockManager(t, nil), nil, expectedEdges, expectedCaps)

}
