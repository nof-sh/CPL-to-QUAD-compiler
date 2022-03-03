package cpq

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

var eof = rune(0)

//lexical token.
type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF
	LPAREN
	RPAREN
	LBRACKET
	RBRACKET
	COMMA
	SEMICOLON
	COLON
	EQUALS
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
	RELOP
	ADDOP
	MULOP
	OR
	AND
	NOT
	ID
	NUM
)

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
	RELOP:      "RELOP",
	ADDOP:      "ADDOP",
	MULOP:      "MULOP",
	OR:         "||",
	AND:        "&&",
	NOT:        "!",
	ID:         "ID",
	NUM:        "NUM",
}

const MaxIdentifierLength = 9

type Scanner struct {
	Reader      *bufio.Reader
	position    Position
	eof         bool
	bufferIndex int
	bufferSize  int
	buffer      [1024]struct {
		ch       rune
		position Position
	}
	DisablePositions bool
}

func (tok TokenType) String() string {
	if tok >= 0 && tok < TokenType(len(tokens)) {
		return tokens[tok]
	}
	return ""
}

func space(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func letter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func digit(ch rune) bool {
	return (ch >= '0' && ch <= '9')
}

func NewScanner(reader io.Reader) *Scanner {
	return &Scanner{
		Reader: bufio.NewReader(reader),
	}
}

// read from bufferred
func (s *Scanner) read() (rune, Position) {
	if s.bufferSize > 0 {
		s.bufferSize--
		return s.curr()
	}
	ch, _, err := s.Reader.ReadRune()
	if err != nil {
		ch = eof
	} else if ch == '\r' {
		if ch, _, err := s.Reader.ReadRune(); err != nil {
		} else if ch != '\n' {
			_ = s.Reader.UnreadRune()
		}
		ch = '\n'
	}
	s.bufferIndex = (s.bufferIndex + 1) % len(s.buffer)
	buffer := &s.buffer[s.bufferIndex]
	buffer.ch, buffer.position = ch, s.position

	if ch == '\n' {
		s.position.Line++
		s.position.Column = 0
	} else if !s.eof {
		s.position.Column++
	}
	if ch == eof {
		s.eof = true
	}

	return s.curr()
}

//returns the last character
func (s *Scanner) curr() (ch rune, pos Position) {
	bufferIndex := (s.bufferIndex - s.bufferSize + len(s.buffer)) % len(s.buffer)
	buffer := &s.buffer[bufferIndex]

	if s.DisablePositions {
		return buffer.ch, Position{}
	}

	return buffer.ch, buffer.position
}

func (s *Scanner) Unscan() {
	s.bufferSize++
}

func (s *Scanner) moveEnd() error {
	for {
		if ch, _ := s.read(); ch == '*' {
		star:
			ch2, _ := s.read()
			if ch2 == '/' {
				return nil
			} else if ch2 == '*' {
				goto star
			} else if ch2 == eof {
				return io.EOF
			}
		} else if ch == eof {
			return io.EOF
		}
	}
}

//Scan returns next token
func (s *Scanner) Scan() Token {

	ch, pos := s.read()
	for {
		if ch == '/' {
			ch2, _ := s.read()
			if ch2 == '*' {
				if err := s.moveEnd(); err != nil {
					return Token{TokenType: ILLEGAL, Lexeme: "", Position: pos}
				}
			} else {
				s.Unscan()
				break
			}
		} else if space(ch) {
			s.findspace()
		} else {
			break
		}
		ch, pos = s.read()
	}
	if letter(ch) {
		s.Unscan()
		return s.findIdentifier()
	} else if digit(ch) {
		s.Unscan()
		return s.findNum()
	}
	switch ch {
	case eof:
		return Token{TokenType: EOF, Lexeme: "EOF", Position: pos}

	case '>', '<':
		ch2, _ := s.read()
		if ch2 == '=' {
			return Token{TokenType: RELOP, Lexeme: string(ch) + string(ch2), Position: pos}
		}
		s.Unscan()
		return Token{TokenType: RELOP, Lexeme: string(ch), Position: pos}

	case '=':
		ch2, _ := s.read()
		if ch2 == '=' {
			return Token{TokenType: RELOP, Lexeme: "==", Position: pos}
		}
		s.Unscan()
		return Token{TokenType: EQUALS, Lexeme: string(ch), Position: pos}

	case '!':
		ch2, _ := s.read()
		if ch2 == '=' {
			return Token{TokenType: RELOP, Lexeme: "!=", Position: pos}
		}
		s.Unscan()
		return Token{TokenType: NOT, Lexeme: string(ch), Position: pos}

	case '|':
		ch2, _ := s.read()
		if ch2 == '|' {
			return Token{TokenType: OR, Lexeme: "||", Position: pos}
		}
		s.Unscan()
		return Token{TokenType: ILLEGAL, Lexeme: string(ch), Position: pos}

	case '&':
		ch2, _ := s.read()
		if ch2 == '&' {
			return Token{TokenType: AND, Lexeme: "&&", Position: pos}
		}
		s.Unscan()
		return Token{TokenType: ILLEGAL, Lexeme: string(ch), Position: pos}

	case '+', '-':
		return Token{TokenType: ADDOP, Lexeme: string(ch), Position: pos}

	case '*', '/':
		return Token{TokenType: MULOP, Lexeme: string(ch), Position: pos}

	case ';':
		return Token{TokenType: SEMICOLON, Lexeme: string(ch), Position: pos}

	case '(':
		return Token{TokenType: LPAREN, Lexeme: string(ch), Position: pos}

	case ')':
		return Token{TokenType: RPAREN, Lexeme: string(ch), Position: pos}

	case '{':
		return Token{TokenType: LBRACKET, Lexeme: string(ch), Position: pos}

	case '}':
		return Token{TokenType: RBRACKET, Lexeme: string(ch), Position: pos}

	case ',':
		return Token{TokenType: COMMA, Lexeme: string(ch), Position: pos}

	case ':':
		return Token{TokenType: COLON, Lexeme: string(ch), Position: pos}
	}

	return Token{TokenType: ILLEGAL, Lexeme: string(ch), Position: pos}
}

func (s *Scanner) findspace() {
	for {
		if ch, _ := s.read(); ch == eof {
			break
		} else if !space(ch) {
			s.Unscan()
			break
		}
	}
}

func (s *Scanner) findIdentifier() Token {
	ch, pos := s.read()

	//Create buffer
	var buf bytes.Buffer
	buf.WriteRune(ch)
	//Read character into the buffer
	for {
		if ch, _ = s.read(); ch == eof {
			break
		} else if !letter(ch) && digit(ch) && ch != '_' {
			s.Unscan()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	switch buf.String() {
	case "break":
		return Token{TokenType: BREAK, Lexeme: buf.String(), Position: pos}
	case "case":
		return Token{TokenType: CASE, Lexeme: buf.String(), Position: pos}
	case "default":
		return Token{TokenType: DEFAULT, Lexeme: buf.String(), Position: pos}
	case "else":
		return Token{TokenType: ELSE, Lexeme: buf.String(), Position: pos}
	case "float":
		return Token{TokenType: FLOAT, Lexeme: buf.String(), Position: pos}
	case "if":
		return Token{TokenType: IF, Lexeme: buf.String(), Position: pos}
	case "input":
		return Token{TokenType: INPUT, Lexeme: buf.String(), Position: pos}
	case "int":
		return Token{TokenType: INT, Lexeme: buf.String(), Position: pos}
	case "output":
		return Token{TokenType: OUTPUT, Lexeme: buf.String(), Position: pos}
	case "switch":
		return Token{TokenType: SWITCH, Lexeme: buf.String(), Position: pos}
	case "while":
		return Token{TokenType: WHILE, Lexeme: buf.String(), Position: pos}
	case "static_cast":
		return Token{TokenType: STATICCAST, Lexeme: buf.String(), Position: pos}
	}
	if len(buf.String()) <= MaxIdentifierLength && !strings.ContainsRune(buf.String(), '_') {
		return Token{TokenType: ID, Lexeme: buf.String(), Position: pos}
	}
	return Token{TokenType: ILLEGAL, Lexeme: buf.String(), Position: pos}
}

func (s *Scanner) findNum() Token {
	var buf bytes.Buffer
	ch, pos := s.read()

	for {
		if !digit(ch) && ch != '.' {
			s.Unscan()
			break
		}
		_, _ = buf.WriteRune(ch)
		ch, _ = s.read()
	}
	return Token{TokenType: NUM, Lexeme: buf.String(), Position: pos}
}
