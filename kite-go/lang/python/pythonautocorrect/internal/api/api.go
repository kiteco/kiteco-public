package api

import "github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"

// ProposalType for a correction to a python file
type ProposalType int

const (
	// None is the zero value for a ProposalType
	None ProposalType = iota
	// Insertion proposal
	Insertion
	// Deletion proposal
	Deletion
)

// Proposal for a correction to a python file.
type Proposal struct {
	Type  ProposalType
	Pos   int
	Token pythonscanner.Token
}

// Proposer proposes possible corrections
// to a file.
type Proposer interface {
	Propose([]pythonscanner.Word) []Proposal
}

// Selector selects a proposal
type Selector interface {
	Select([]Proposal) Proposal
}
