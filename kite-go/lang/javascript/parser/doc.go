package parser

// All types prepended with godoc are meant to serve as headers for the
// go doc and do not represent real types.

// To generate the parser run
//   1) install goimports https://godoc.org/golang.org/x/tools/cmd/goimports
//   2) install pigeon https://godoc.org/golang.org/x/tools/cmd/goimports
//   3) run `make clean && make all`

// - Arguments currently include () as part of their literal string
// - `BinaryExpression`s and `LogicalExpression`s contain their operators as part of their literal string
// - `BinaryExpression`s and `LogicalExpression`s are completely collapsed, precedence is completely ignored
// - The LHS of `AssignmentExpression`s ("left") is still quite liberal and uses the es5 specification
//   instead of the "pattern" specification for es2015
//   SEE: "FIXME: This describes the Esprima and Acorn behaviors, which is not currently aligned with the SpiderMonkey behavior."
//         IN: https://github.com/estree/estree/blob/master/es2015.md#expressions
//   AND: https://github.com/estree/estree/pull/20#issuecomment-74584758
// - We ignore the return value of a `YieldExpression`
//   SEE: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/yield
// - Non-BMP characters are completely ignored to avoid surrogate pair
//   handling.
// - One can create identifiers containing illegal characters using Unicode
//   escape sequences. For example, "abcd\u0020efgh" is not a valid
//   identifier, but it is accepted by the parser.
// - Strict mode is not recognized.
// - Partial handling of TemplateLiterals
//   SEE: https://github.com/estree/estree/blob/master/es2015.md#template-literals
//   SEE: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Template_literals
// - Class method names are allowed to be identifiers, this is technically not allowed by js.
type godocTODO int
