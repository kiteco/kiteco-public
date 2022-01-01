package lexicalproviders

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

func eventFromBuffer(b data.SelectedBuffer, path string) *component.EditorEvent {
	return &component.EditorEvent{
		Text: b.Text(),
		Selections: []*component.Selection{
			&component.Selection{
				Start: int64(b.Selection.End),
				End:   int64(b.Selection.End),
			},
		},
		Filename: path,
	}
}

type curatedContextTC struct {
	src1     string
	src2     string
	expected data.Completion
}

func TestGolang_CuratedContext(t *testing.T) {
	tcs := []curatedContextTC{
		curatedContextTC{
			src1: `
		package state

		type Nodes$ struct`,
			src2: `
		package state

		type State struct {
			EmbeddedContext   []int
			UnembeddedContext []int
			PredictedContext  [][]int
			Prefix            string
			Rand              *rand.Rand

			Search SearchConfig

			InitialPredictions []Predicted
			Expansions         []Expansion
			LatestPredictions  []Predicted

			PRM *PartialRunModel

			depth       int // For internal bookkeeping
			incremental chan Predicted
			vocabIDs    []int
		}

		func toStrings(ctx []int) []string {
			strs := make([]string, 0, len(ctx))
			for _, c := range ctx {
				strs = append(strs, strconv.Itoa(c))
			}
			return strs
		}

		func (n *N$`,
			expected: data.Completion{
				Snippet: data.NewSnippet("Nodes"),
				Replace: data.Selection{Begin: -1, End: 0},
			},
		},
		curatedContextTC{
			src1: `
package evaluate

func (e *Evaluator) Clear$() {
}
`,
			src2: `
package evaluate

type State struct {
	EmbeddedContext   []int
	UnembeddedContext []int
	PredictedContext  [][]int
	Prefix            string
	Rand              *rand.Rand

	Search SearchConfig

	InitialPredictions []Predicted
	Expansions         []Expansion
	LatestPredictions  []Predicted

	PRM *PartialRunModel

	depth       int // For internal bookkeeping
	incremental chan Predicted
	vocabIDs    []int
}

func (e *Evaluator) Evaluate() {
	e.Cl$
`,
			expected: data.Completion{
				Snippet: data.NewSnippet("Clear"),
				Replace: data.Selection{Begin: -2, End: 0},
			},
		},
	}

	for i, tc := range tcs {
		initModels(t, lexicalmodels.DefaultModelOptions)
		editorEvents := []*component.EditorEvent{
			eventFromBuffer(processTemplate(t, tc.src1), "./src1.go"),
			eventFromBuffer(processTemplate(t, tc.src2), "./src2.go"),
		}
		res, err := runWithEditorEvents(t, Text{}, tc.src2, "./src2.go", editorEvents)
		require.NoError(t, err)
		require.True(t, res.containsRoot(tc.expected), "test case %d", i)
	}
}

func TestJavascript_CuratedContext(t *testing.T) {
	tcs := []curatedContextTC{
		curatedContextTC{
			src1: "this.restartKite()$",
			src2: `
		class Navigation extends React.Component {
		  render() {
		    const { index, length, entries } = this.props.history
		    const showGoBack = (index > 0) &&
		          (entries[index - 1].pathname.startsWith('/docs/') ||
		          entries[index - 1].pathname.startsWith('/examples/')),

		          showGoForward = (index < length - 1) &&
		          (entries[index + 1].pathname.startsWith('/docs/') ||
		          entries[index + 1].pathname.startsWith('/examples/'))

		    return (
		      <div className="navigation__history">
		        <div className={"navigation__history__back " + (showGoBack ? 'navigation__history__button--enabled' : '')} onClick={this.props.goBack}></div>
		        <div className={"navigation__history__forward " + (showGoForward ? 'navigation__history__button--enabled' : '')} onClick={this.props.goForward}></div>
		      </div>
		    )
		  }
		}

		const mapStateToProps = (state, ownProps) => ({
		  ...ownProps,
		})

		const mapDispatchToProps = dispatch => ({
		  goBack: params => dispatch(goBack()),
		  goForward: params => dispatch(goForward()),
		})

		enableKiteGo = () => {
		  this.props.setKiteGoEnabled(true).then(() => {
		    track({ event: 'copilot_settings_kite_lexical_enabled' })
		      this.r$
		`,
			expected: data.Completion{
				Snippet: data.NewSnippet("restartKite"),
				Replace: data.Selection{Begin: -1, End: 0},
			},
		},
		curatedContextTC{
			src1: `
import file-tree from './assets/file-tree.css'$
`,
			src2: `
class Navigation extends React.Component {
  render() {
    const { index, length, entries } = this.props.history
    const showGoBack = (index > 0) &&
          (entries[index - 1].pathname.startsWith('/docs/') ||
          entries[index - 1].pathname.startsWith('/examples/')),

          showGoForward = (index < length - 1) &&
          (entries[index + 1].pathname.startsWith('/docs/') ||
          entries[index + 1].pathname.startsWith('/examples/'))

    return (
      <div className="navigation__history">
        <div className={"navigation__history__back " + (showGoBack ? 'navigation__history__button--enabled' : '')} onClick={this.props.goBack}></div>
        <div className={"navigation__history__forward " + (showGoForward ? 'navigation__history__button--enabled' : '')} onClick={this.props.goForward}></div>
      </div>
    )
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
})

const mapDispatchToProps = dispatch => ({
  goBack: params => dispatch(goBack()),
  goForward: params => dispatch(goForward()),
})

const FileTree = ({
  root,
  separator,
  defaultOpen,
}) =>
  <div className='f$
`,
			expected: data.Completion{
				Snippet: data.NewSnippet("file-tree"),
				Replace: data.Selection{Begin: -1, End: 0},
			},
		},
		curatedContextTC{
			src1: "import { withRouter } from 'react-router'$",
			src2: `
class Navigation extends React.Component {
  render() {
    const { index, length, entries } = this.props.history
    const showGoBack = (index > 0) &&
          (entries[index - 1].pathname.startsWith('/docs/') ||
          entries[index - 1].pathname.startsWith('/examples/')),

          showGoForward = (index < length - 1) &&
          (entries[index + 1].pathname.startsWith('/docs/') ||
          entries[index + 1].pathname.startsWith('/examples/'))

    return (
      <div className="navigation__history">
        <div className={"navigation__history__back " + (showGoBack ? 'navigation__history__button--enabled' : '')} onClick={this.props.goBack}></div>
        <div className={"navigation__history__forward " + (showGoForward ? 'navigation__history__button--enabled' : '')} onClick={this.props.goForward}></div>
      </div>
    )
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
})

const mapDispatchToProps = dispatch => ({
  goBack: params => dispatch(goBack()),
  goForward: params => dispatch(goForward()),
})

export default w$
`,
			expected: data.Completion{
				Snippet: data.NewSnippet("withRouter"),
				Replace: data.Selection{Begin: -1, End: 0},
			},
		},
	}
	for i, tc := range tcs {
		initModels(t, lexicalmodels.DefaultModelOptions)
		editorEvents := []*component.EditorEvent{
			eventFromBuffer(processTemplate(t, tc.src1), "./src1.js"),
			eventFromBuffer(processTemplate(t, tc.src2), "./src2.js"),
		}
		res, err := runWithEditorEvents(t, Text{}, tc.src2, "./src2.js", editorEvents)
		require.NoError(t, err)
		require.True(t, res.containsRoot(tc.expected), "test case %d", i)
	}
}
