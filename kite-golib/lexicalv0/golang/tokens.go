package golang

import "go/token"

var (
	// AllTokens ...
	AllTokens = []token.Token{
		// Special tokens
		token.ILLEGAL,
		token.EOF,
		token.COMMENT,

		// Identifiers and basic type literals
		// (these tokens stand for classes of literals)

		// token.IDENT,  // main -> Idents are encoded separately via BPE
		token.INT,    // 12345
		token.FLOAT,  // 123.45
		token.IMAG,   // 123.45i
		token.CHAR,   // 'a'
		token.STRING, // "abc"

		// Operators and delimiters
		token.ADD, // +
		token.SUB, // -
		token.MUL, // *
		token.QUO, // /
		token.REM, // %

		token.AND,     // &
		token.OR,      // |
		token.XOR,     // ^
		token.SHL,     // <<
		token.SHR,     // >>
		token.AND_NOT, // &^

		token.ADD_ASSIGN, // +=
		token.SUB_ASSIGN, // -=
		token.MUL_ASSIGN, // *=
		token.QUO_ASSIGN, // /=
		token.REM_ASSIGN, // %=

		token.AND_ASSIGN,     // &=
		token.OR_ASSIGN,      // |=
		token.XOR_ASSIGN,     // ^=
		token.SHL_ASSIGN,     // <<=
		token.SHR_ASSIGN,     // >>=
		token.AND_NOT_ASSIGN, // &^=

		token.LAND,  // &&
		token.LOR,   // ||
		token.ARROW, // <-
		token.INC,   // ++
		token.DEC,   // --

		token.EQL,    // ==
		token.LSS,    // <
		token.GTR,    // >
		token.ASSIGN, // =
		token.NOT,    // !

		token.NEQ,      // !=
		token.LEQ,      // <=
		token.GEQ,      // >=
		token.DEFINE,   // :=
		token.ELLIPSIS, // ...

		token.LPAREN, // (
		token.LBRACK, // [
		token.LBRACE, // {
		token.COMMA,  // ,
		token.PERIOD, // .

		token.RPAREN,    // )
		token.RBRACK,    // ]
		token.RBRACE,    // }
		token.SEMICOLON, // ;
		token.COLON,     // :

		// Keywords
		token.BREAK,
		token.CASE,
		token.CHAN,
		token.CONST,
		token.CONTINUE,

		token.DEFAULT,
		token.DEFER,
		token.ELSE,
		token.FALLTHROUGH,
		token.FOR,

		token.FUNC,
		token.GO,
		token.GOTO,
		token.IF,
		token.IMPORT,

		token.INTERFACE,
		token.MAP,
		token.PACKAGE,
		token.RANGE,
		token.RETURN,

		token.SELECT,
		token.STRUCT,
		token.SWITCH,
		token.TYPE,
		token.VAR,
	}

	// TokenToIdx ...
	TokenToIdx map[token.Token]int

	// IdxToToken ...
	IdxToToken map[int]token.Token
)

func init() {
	TokenToIdx, IdxToToken = tokenMap(AllTokens)
}

func tokenMap(tokens []token.Token) (map[token.Token]int, map[int]token.Token) {
	m1 := make(map[token.Token]int)
	m2 := make(map[int]token.Token)
	for idx, token := range tokens {
		m1[token] = idx
		m2[idx] = token
	}
	return m1, m2
}
