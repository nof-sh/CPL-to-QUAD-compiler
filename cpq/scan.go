package lexer

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

//import (
//	"bufio"
//	"bytes"
//	"io"
//	"strings"
//)

// MaxIdentifierLength is the maximum length of IDs in CPL.
const MaxIdentifierLength = 9

// Scanner represents a lexical scanner.
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
	DisablePositions bool // for testing.
}

// NewScanner returns a new instance of Scanner.
func NewScanner(reader io.Reader) *Scanner {
	return &Scanner{
		Reader: bufio.NewReader(reader),
	}
}

// read reads the next rune from the bufferred reader.
// Returns the rune(0) if an error occurs (or io.EOF is returned).
func (s *Scanner) read() (rune, Position) {
	// If we have unread characters then read them off the buffer first.
	if s.bufferSize > 0 {
		s.bufferSize--
		return s.curr()
	}

	// Read next rune from underlying reader.
	// Any error (including io.EOF) should return as EOF.
	ch, _, err := s.Reader.ReadRune()
	if err != nil {
		ch = eof
	} else if ch == '\r' {
		if ch, _, err := s.Reader.ReadRune(); err != nil {
			// nop
		} else if ch != '\n' {
			_ = s.Reader.UnreadRune()
		}
		ch = '\n'
	}

	// Save character and position to the buffer.
	s.bufferIndex = (s.bufferIndex + 1) % len(s.buffer)
	buffer := &s.buffer[s.bufferIndex]
	buffer.ch, buffer.position = ch, s.position

	// Update position.
	// Only count EOF once.
	if ch == '\n' {
		s.position.Line++
		s.position.Column = 0
	} else if !s.eof {
		s.position.Column++
	}

	// Mark the reader as EOF.
	// This is used so we don't double count EOF characters.
	if ch == eof {
		s.eof = true
	}

	return s.curr()
}

// curr returns the last read character and position.
func (s *Scanner) curr() (ch rune, pos Position) {
	bufferIndex := (s.bufferIndex - s.bufferSize + len(s.buffer)) % len(s.buffer)
	buffer := &s.buffer[bufferIndex]

	if s.DisablePositions {
		return buffer.ch, Position{}
	}

	return buffer.ch, buffer.position
}

// Unscan pushes the previously token back onto the buffer.
func (s *Scanner) Unscan() {
	s.bufferSize++
}

// Scan returns the next token and literal value.
func (s *Scanner) Scan() Token {
	// Read the next rune.
	ch, pos := s.read()

	// Skip comments and whitespaces.
	for {
		if ch == '/' {
			ch2, _ := s.read()
			if ch2 == '*' {
				if err := s.skipUntilEndComment(); err != nil {
					return Token{TokenType: ILLEGAL, Lexeme: "", Position: pos}
				}
			} else {
				s.Unscan()
				break
			}
		} else if isWhitespace(ch) {
			s.scanWhitespace()
		} else {
			break
		}

		ch, pos = s.read()
	}

	// If we see a letter then consume as an ID or reserved word.
	if isLetter(ch) {
		s.Unscan()
		return s.scanIdentifier()
	} else if isDigit(ch) {
		s.Unscan()
		return s.scanNumber()
	}

	// Otherwise read the individual character.
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

// scanWhitespace consumes the current rune and all contiguous whitespace.
func (s *Scanner) scanWhitespace() {
	// Read every subsequent whitespace character into the buffer.
	// Non-whitespace characters and EOF will cause the loop to exit.
	for {
		if ch, _ := s.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.Unscan()
			break
		}
	}
}

// scanIdentifier consumes the current rune and all contiguous identifier runes.
func (s *Scanner) scanIdentifier() Token {
	ch, pos := s.read()

	// Create a buffer and read the current character into it.
	var buf bytes.Buffer
	buf.WriteRune(ch)

	// Read every subsequent ident character into the buffer.
	// Non-ident characters and EOF will cause the loop to exit.
	for {
		if ch, _ = s.read(); ch == eof {
			break
		} else if !isLetter(ch) && !isDigit(ch) && ch != '_' {
			s.Unscan()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}

	// If the string matches a keyword then return that keyword.
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

	// Otherwise return as a regular identifier - just need to make sure its length is okay
	// and it doesn't contain an underscore, which is an illegal character for IDs.
	if len(buf.String()) <= MaxIdentifierLength && !strings.ContainsRune(buf.String(), '_') {
		return Token{TokenType: ID, Lexeme: buf.String(), Position: pos}
	}

	return Token{TokenType: ILLEGAL, Lexeme: buf.String(), Position: pos}
}

// scanNumber consumes a contiguous series of digits.
func (s *Scanner) scanNumber() Token {
	var buf bytes.Buffer
	ch, pos := s.read()

	for {
		if !isDigit(ch) && ch != '.' {
			s.Unscan()
			break
		}
		_, _ = buf.WriteRune(ch)
		ch, _ = s.read()
	}

	return Token{TokenType: NUM, Lexeme: buf.String(), Position: pos}
}

// skipUntilEndComment skips characters until it reaches a '*/' symbol.
func (s *Scanner) skipUntilEndComment() error {
	for {
		if ch, _ := s.read(); ch == '*' {
			// We might be at the end.
		star:
			ch2, _ := s.read()
			if ch2 == '/' {
				return nil
			} else if ch2 == '*' {
				// We are back in the state machine since we see a star.
				goto star
			} else if ch2 == eof {
				return io.EOF
			}
		} else if ch == eof {
			return io.EOF
		}
	}
}
