package mutatetest

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

func printNode(n pythonast.Node) string {
	var buf bytes.Buffer
	pythonast.Print(n, &buf, "\t")
	return buf.String()
}

func assertAST(t *testing.T, expected string, actual string) {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)

	if actual != expected {
		expectedLines := strings.Split(expected, "\n")
		actualLines := strings.Split(actual, "\n")

		n := len(expectedLines)
		if len(actualLines) > n {
			n = len(actualLines)
		}

		errorLine := -1
		sidebyside := fmt.Sprintf("      | %-40s | %-40s |\n", "EXPECTED", "ACTUAL")
		var errorExpected, errorActual string
		for i := 0; i < n; i++ {
			var expectedLine, actualLine string
			if i < len(expectedLines) {
				expectedLine = strings.Replace(expectedLines[i], "\t", "    ", -1)
			}
			if i < len(actualLines) {
				actualLine = strings.Replace(actualLines[i], "\t", "    ", -1)
			}
			symbol := "   "
			if actualLine != expectedLine {
				symbol = "***"
				if errorLine == -1 {
					errorLine = i
					errorExpected = strings.TrimSpace(expectedLine)
					errorActual = strings.TrimSpace(actualLine)
				}
			}
			sidebyside += fmt.Sprintf("%-6s| %-40s | %-40s |\n", symbol, expectedLine, actualLine)
		}

		t.Errorf("expected %s but got %s (line %d):\n%s", errorExpected, errorActual, errorLine, sidebyside)
	}

	t.Log("\n" + actual)
}

func requireParsed(t *testing.T, src string) *pythonast.Module {
	mod, err := pythonparser.Parse(kitectx.Background(), []byte(src), pythonparser.Options{})
	require.NoError(t, err, "got parse error: %v", err)
	require.NotNil(t, mod, "got nil module after parsing")
	return mod
}
