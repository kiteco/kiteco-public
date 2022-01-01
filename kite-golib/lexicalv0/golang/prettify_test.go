package golang

import (
	"bytes"
	"strings"
	"testing"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/golang"
)

var defaultConfig = Config{
	Indent:          "\t",
	SpaceAfterComma: true,
}

func TestPrettify_Imports(t *testing.T) {
	case1 := `import "fmt"`
	case2 := `import sitter "github.com/kiteco/go-tree-sitter"`
	case3 := `
import (
	"fmt"
	sitter "github.com/kiteco/go-tree-sitter"
)`
	runPrettifyCase(t, defaultConfig, case1, case1)
	runPrettifyCase(t, defaultConfig, case2, case2)
	runPrettifyCase(t, defaultConfig, case3, case3)
}

func TestPrettify_TopLevelDecls(t *testing.T) {
	case1 := `
var foo int = 5
var cloud, sun string = "none"
const bar []int = []int{4, 5, 6}
var baz [][]int = []int{{1, 2, 3}, {4, 5, 6}}
const bat map[string]bool = map[string]bool{
	"bat": true,
}
const (
	baz string = "baz"
	bat string = "bat"
)
type Animal struct {
	Name string
	Size int
}
type (
	Cat struct {
		Head int
		Tail int
		Info []string
	}
	Dog struct {
		Head int
		Tail int
	}
)
type TemplateSet interface {
	Render(w io.Writer, name string, payload interface{}) error
}
type PlanID = string
type Plans []PlanID
type Empty struct {}
func main() {
	args := struct {
		name string
		id int
	}{
		name: "name",
		id: "id",
	}
}`
	runPrettifyCase(t, defaultConfig, case1, case1)
}

func TestPrettify_ForStatement(t *testing.T) {
	case1 := `
func main() {
	for _, p := range pieces {
		for i := 0; i < p.length; i++ {
			continue
		}
		for 5 < i {
			i = 5
		}
		if err != nil {
			fmt.Println(err)
		}
	}
}`
	runPrettifyCase(t, defaultConfig, case1, case1)
}

func TestPrettify_FuncDecls(t *testing.T) {
	case1 := `
func (c *Cat) Feed(food, water int) (full bool, _ error) {
	c.Food += food
	c.Water += water
	return true, nil
}
func Printf(format string, a ...interface{}) (n int, err error) {
	return Fprintf(os.Stdout, format, a...)
}
func LastIndexFunc(s string, f func(rune) bool) int {
	return lastIndexFunc(s, f, true)
}
func Positional(vs ...Value) Args {
	return Args{
		Positional: vs,
	}
}
func FormatCompletion(input string, c data.Completion, config Config, match render.MatchOption) data.Completion {
	return render.FormatCompletion(input, c, python.GetLanguage(), match, func(w io.Writer, src []byte, n *sitter.Node) ([]render.OffsetMapping, error) {
		return Prettify(w, config, src, c.Replace.Begin, n)
	})
}`
	runPrettifyCase(t, defaultConfig, case1, case1)
}

func TestPrettify_Select(t *testing.T) {
	case1 := `
func fibonacci(c, quit chan int) {
	x, y := 0, 1
	for {
		select {
		case empty:
		case c <- x:
			x, y = y, x
		case <-quit:
			fmt.Println("quit")
			return
		default:
			return
		}
	}
}`
	runPrettifyCase(t, defaultConfig, case1, case1)
}

func TestPrettify_Switch(t *testing.T) {
	case1 := `
func render() {
	// General cases, merge if both sides agree
	switch lastRight {
	case must:
		merged = append(merged, ph(curr))
	case never:
		if curr == "\n" {
			merged = append(merged, " ", ph(curr))
		}
	case can:
		switch lookup[currToken].left {
		case must, can:
			merged = append(merged, ph(curr))
		case never:
			merged = append(merged, " ", ph(curr))
		}
	}
	switch annotation := annotation.(type) {
	case *ImageAnnotation, *PictureAnnotation:
		usedFiles[annotation.Path] = struct {}{}
	case *FileAnnotation:
		usedFiles[annotation.Path] = struct {}{}
	}
}
`
	runPrettifyCase(t, defaultConfig, case1, case1)
}

func TestPrettify_Slice(t *testing.T) {
	case1 := `
func main() {
	foo = bar[1:5]
	foo = bar[i:]
	foo = bar[i+1:]
	foo = bar[i-1 : i+1]
}`
	runPrettifyCase(t, defaultConfig, case1, case1)
}

func TestPrettify_Labeled(t *testing.T) {
	case1 := `
func main() {
	x := [][]int{}
outer:
	for i := range x {
		for j := range i {
			break outer
		}
	}
}`
	runPrettifyCase(t, defaultConfig, case1, case1)
}

func TestPrettify_Comments(t *testing.T) {
	case1 := `
func main() {
	// First line
	// Second line
	var a int // Variable declaration
}`
	runPrettifyCase(t, defaultConfig, case1, case1)
}

func TestPrettify_Depth(t *testing.T) {
	case1 := `
func loadAnnotations(output *sandbox.Result) ([]OutputSegment, error) {
	for _, blob := range blobs {
		switch blob.Type {
		case outputBlob:
			annotations = append(annotations, &PlaintextAnnotation{
				Line: line,
				Value: blob.Content,
			})
		case emitBlob:
			annotation, err := parseAnnotation([]byte(blob.Content), line)
		}
	}
	return annotations, nil
}`
	runPrettifyCase(t, defaultConfig, case1, case1)
}

func runPrettifyCase(t *testing.T, conf Config, src, want string) {
	src, want = strings.TrimSpace(src), strings.TrimSpace(want)

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(golang.GetLanguage())
	b := []byte(src)
	tree := parser.Parse(b)
	defer tree.Close()

	var buf bytes.Buffer
	if _, err := Prettify(&buf, conf, b, 0, len(b), tree.RootNode()); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != want {
		t.Fatalf("want:\n%s\ngot:\n%s\n", want, got)
	}
}
