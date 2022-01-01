package ast

// Terminal node in the ast.
type Terminal string

const (
	// Unknown terminal node.
	Unknown Terminal = "<unknown>"
	// Equals terminal node.
	Equals = "="
	// Colon terminal node.
	Colon = ":"
)
