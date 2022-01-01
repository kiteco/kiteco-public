package pythonautocorrect

import (
	"fmt"
	"strings"

	"github.com/kiteco/kiteco/kite-go/autocorrect"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonautocorrect/internal/api"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonautocorrect/internal/options"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonautocorrect/internal/proposers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonautocorrect/internal/selectors"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const version = uint64(1)

// ProposerType specifies the types of proposers available.
type ProposerType string

const (
	// ParserBased proposer
	ParserBased ProposerType = "parser_based"
)

// SelectorType specifies the types of the selectors available
type SelectorType string

const (
	// DefaultSelector is the default selector
	DefaultSelector SelectorType = "default_selector"
)

// Options for the correcter
type Options struct {
	Proposer ProposerType
	Selector SelectorType
}

// DefaultOptions for the suggester
var DefaultOptions = Options{
	Proposer: ParserBased,
	Selector: DefaultSelector,
}

// Correcter for autocorrect.
type Correcter struct {
	opts Options
}

// ensure we implement the Correcter interface: if not, the type checker will complain when building pythonautocorrect
var _ = autocorrect.Correcter((*Correcter)(nil))

// NewCorrecter for corrections.
func NewCorrecter(opts Options) *Correcter {
	return &Correcter{
		opts: opts,
	}
}

// Version of the correcter
func (c *Correcter) Version() uint64 {
	return version
}

// Correct file.
func (c *Correcter) Correct(ctx kitectx.Context, uid int64, mid string, req editorapi.AutocorrectRequest) (autocorrect.Corrections, error) {
	ctx.CheckAbort()

	words, err := pythonscanner.Lex([]byte(req.Buffer), options.Lex)

	if err != nil {
		return autocorrect.Corrections{}, fmt.Errorf("lex error: %v", err)
	}

	// parse to check if the file actually has an error
	_, parseErr := pythonparser.ParseWords(ctx, []byte(req.Buffer), words, options.Parse)
	parseErrs, _ := parseErr.(errors.Errors)

	proposals := c.propose(words, parseErrs)
	if len(proposals) == 0 {
		return autocorrect.Corrections{
			NewBuffer: req.Buffer,
		}, nil
	}

	selected := c.selectProposal(proposals)
	if selected.Type == api.None {
		return autocorrect.Corrections{
			NewBuffer: req.Buffer,
		}, nil
	}

	var buffer string
	switch {
	case parseErr == nil:
		buffer = req.Buffer // never suggest a correction if the file was syntactically valid
	case selected.Type == api.Insertion:
		buffer = strings.Join([]string{
			req.Buffer[:selected.Pos],
			selected.Token.String(),
			req.Buffer[selected.Pos:],
		}, "")
	default:
		buffer = strings.Join([]string{
			req.Buffer[:selected.Pos],
			req.Buffer[selected.Pos+1:],
		}, "")
	}

	if buffer != req.Buffer {
		_, err := pythonparser.Parse(ctx, []byte(buffer), options.Parse)
		if err != nil {
			// never suggest a correction if the resulting file is not syntactically valid
			buffer = req.Buffer
		}
	}

	return autocorrect.Corrections{
		NewBuffer: buffer,
	}, nil
}

func (c *Correcter) propose(words []pythonscanner.Word, parseErr errors.Errors) []api.Proposal {
	switch c.opts.Proposer {
	case ParserBased:
		return proposers.NewParserBased(parseErr).Propose(words)
	default:
		panic(fmt.Sprintf("invalid proposer option: %v", c.opts.Proposer))
	}
}

func (c *Correcter) selectProposal(proposals []api.Proposal) api.Proposal {
	switch c.opts.Selector {
	case DefaultSelector:
		return selectors.DefaultSelector(proposals)
	default:
		panic(fmt.Sprintf("invalid selector option: %v", c.opts.Selector))
	}
}
