package lexar

// Token represents a lexical token.
type TokenType int

// CPL's tokens
const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF

	// Symbols
	LPAREN    // (
	RPAREN    // )
	LBRACKET  // {
	RBRACKET  // }
	COMMA     // ,
	SEMICOLON // ;
	COLON     // :
	EQUALS    // =

	// Keywords
	BREAK
	CASE
	DEFAULT
	ELSE
	FLOAT
	IF
	INPUT
	INT
	OUTPUT
	STATICCAST
	SWITCH
	WHILE

	// Operators
	RELOP // == | != | < | > | >= | <=
	ADDOP // + | -
	MULOP // * | /
	OR    // ||
	AND   // &&
	NOT   // !

	// Literals
	ID
	NUM
)

// Position specifies the line and character position of a token.
// The Column and Line are both zero-based indexes.
type Position struct {
	Line   int
	Column int
}

type Token struct {
	TokenType TokenType
	Lexeme    string
	Position  Position
}

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",

	// Symbols
	LPAREN:    "(",
	RPAREN:    ")",
	LBRACKET:  "{",
	RBRACKET:  "}",
	COMMA:     ",",
	SEMICOLON: ";",
	COLON:     ":",
	EQUALS:    "=",

	// Keywords
	BREAK:      "break",
	CASE:       "case",
	DEFAULT:    "default",
	ELSE:       "else",
	FLOAT:      "float",
	IF:         "if",
	INPUT:      "input",
	INT:        "int",
	OUTPUT:     "output",
	STATICCAST: "static_cast",
	SWITCH:     "switch",
	WHILE:      "while",

	// Operators
	RELOP: "RELOP",
	ADDOP: "ADDOP",
	MULOP: "MULOP",
	OR:    "||",
	AND:   "&&",
	NOT:   "!",

	// Literals
	ID:  "ID",
	NUM: "NUM",
}

// String returns the string representation of the token.
func (tok TokenType) String() string {
	if tok >= 0 && tok < TokenType(len(tokens)) {
		return tokens[tok]
	}
	return ""
}
