package cpq

import (
	"fmt"
	"strconv"
	"strings"
)

type ErrorType struct {
	Message  string
	Found    string
	Expected []string
	Pos      Position
}

//CPL parser.
type Parser struct {
	Errors    []ErrorType
	scanner   *Scanner
	lookahead Token
}

//returns the string of the error
func (e *ErrorType) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s at line %d, char %d", e.Message, e.Pos.Line+1, e.Pos.Column+1)
	}
	return fmt.Sprintf("found %s, expected %s at line %d, char %d", e.Found,
		strings.Join(e.Expected, ", "), e.Pos.Line+1, e.Pos.Column+1)
}

//returns ParseError
func newError(found string, expected []string, pos Position) ErrorType {
	return ErrorType{
		Message:  "",
		Found:    found,
		Expected: expected,
		Pos:      pos,
	}
}

func (p *Parser) addError(e ErrorType) {
	for _, err := range p.Errors {
		if err.Pos == e.Pos {
			return
		}
	}
	p.Errors = append(p.Errors, e)
}

//returns new parser
func NewParser(scanner *Scanner) *Parser {
	return &Parser{
		Errors:    []ErrorType{},
		scanner:   scanner,
		lookahead: scanner.Scan(),
	}
}

func Parse(s string) (*Program, []ErrorType) {
	parser := NewParser(NewScanner(strings.NewReader(s)))
	return parser.ParseProgram(), parser.Errors
}

func (p *Parser) matchToken(tokenTypes ...TokenType) (*Token, bool) {
	for _, tokType := range tokenTypes {
		if tokType == p.lookahead.TokenType {
			token := p.lookahead
			p.lookahead = p.scanner.Scan()
			return &token, true
		}
	}
	return &p.lookahead, false
}

func (p *Parser) match(tokenTypes ...TokenType) (*Token, bool) {
	if token, ok := p.matchToken(tokenTypes...); ok {
		return token, true
	}
	return &p.lookahead, false
}

func (p *Parser) skip() {
	p.lookahead = p.scanner.Scan()
}

// 	program -> declarations stmt_block
func (p *Parser) ParseProgram() *Program {
	program := &Program{Pos: p.lookahead.Position}
	program.Declarations = p.ParseDeclarations()
	program.StatementsBlock = p.StatementsBlock()
	// check for EOF at the file
	if token, ok := p.match(EOF); !ok {
		p.addError(newError(token.Lexeme, []string{"EOF"}, program.Pos))
	}
	return program
}

// 	declarations -> declaration declarations | ε
func (p *Parser) ParseDeclarations() []Declaration {
	declarations := []Declaration{}
	for p.lookahead.TokenType == ID {
		declarations = append(declarations, *p.ParseDeclaration())
	}

	return declarations
}

// 	declaration -> idlist ':' type ';'
func (p *Parser) ParseDeclaration() *Declaration {
	declaration := &Declaration{Pos: p.lookahead.Position}
	declaration.Names = p.ParseIDList()

	if token, ok := p.match(COLON); !ok {
		p.addError(newError(token.Lexeme, []string{":"}, token.Position))
	}
	declaration.Type = p.ParseType()
	if token, ok := p.match(SEMICOLON); !ok {
		p.addError(newError(token.Lexeme, []string{";"}, token.Position))
	}
	return declaration
}

// 	type -> INT | FLOAT
func (p *Parser) ParseType() DataType {
	token, ok := p.match(INT, FLOAT)
	if !ok {
		p.skip()
		p.addError(newError(token.Lexeme, []string{"int", "float"}, token.Position))
		return Unknown
	}
	switch token.TokenType {
	case INT:
		return Integer
	case FLOAT:
		return Float
	}
	return Unknown
}

// 	idlist -> ID idlist'
// 	idlist' -> ',' ID idlist' | ε
func (p *Parser) ParseIDList() []string {
	names := []string{}
	// Parse the first name
	if token, ok := p.match(ID); ok {
		names = append(names, token.Lexeme)
	} else {
		p.addError(newError(token.Lexeme, []string{"ID"}, token.Position))
	}
	// Parse other names if exist
	for p.lookahead.TokenType == COMMA {
		p.match(COMMA)

		if token, ok := p.match(ID); ok {
			names = append(names, token.Lexeme)
		} else {
			p.addError(newError(token.Lexeme, []string{"ID"}, token.Position))
		}
	}
	return names
}

//	stmt -> assignment_stmt | input_stmt | output_stmt | if_stmt | while_stmt| switch_stmt | break_stmt | stmt_block
func (p *Parser) Statement() Statement {
	switch p.lookahead.TokenType {
	case ID:
		return p.AssignmentStatement()

	case INPUT:
		return p.InputStatement()

	case OUTPUT:
		return p.OutputStatement()

	case IF:
		return p.IfStatement()

	case WHILE:
		return p.WhileStatement()

	case SWITCH:
		return p.SwitchStatement()

	case BREAK:
		return p.BreakStatement()

	case LBRACKET:
		return p.StatementsBlock()
	}
	return nil
}

// 	assignment_stmt -> ID '=' assignment_stmt'
// 	assignment_stmt' -> expression ';'| STATIC_CAST '(' type ')' '(' expression ')' ';
func (p *Parser) AssignmentStatement() *Assignment {
	result := &Assignment{Pos: p.lookahead.Position}

	if token, ok := p.match(ID); ok {
		result.Variable = token.Lexeme
	} else {
		p.addError(newError(token.Lexeme, []string{"ID"}, token.Position))
	}
	if token, ok := p.match(EQUALS); !ok {
		p.addError(newError(token.Lexeme, []string{"ID"}, token.Position))
	}
	if p.lookahead.TokenType == STATICCAST {
		p.match(STATICCAST)

		if token, ok := p.match(LPAREN); !ok {
			p.addError(newError(token.Lexeme, []string{"("}, token.Position))
		}
		result.CastType = p.ParseType()

		if token, ok := p.match(RPAREN); !ok {
			p.addError(newError(token.Lexeme, []string{")"}, token.Position))
		}
	}
	if token, ok := p.match(SEMICOLON); !ok {
		p.addError(newError(token.Lexeme, []string{";"}, token.Position))
	}
	return result
}

// 	input_stmt -> INPUT '(' ID ')' ';'
func (p *Parser) InputStatement() *Input {
	if _, ok := p.match(INPUT); !ok {
		return nil
	}

	result := &Input{Pos: p.lookahead.Position}

	if token, ok := p.match(LPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{"("}, token.Position))
	}
	if token, ok := p.match(ID); ok {
		result.Variable = token.Lexeme
	} else {
		p.addError(newError(token.Lexeme, []string{"ID"}, token.Position))
	}
	if token, ok := p.match(RPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{")"}, token.Position))
	}
	if token, ok := p.match(SEMICOLON); !ok {
		p.addError(newError(token.Lexeme, []string{";"}, token.Position))
	}
	return result
}

// 	output_stmt -> OUTPUT '(' expression ')' ';'
func (p *Parser) OutputStatement() *Output {
	if _, ok := p.match(OUTPUT); !ok {
		return nil
	}
	result := &Output{Position: p.lookahead.Position}

	if token, ok := p.match(LPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{"("}, token.Position))
	}
	if token, ok := p.match(RPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{")"}, token.Position))
	}
	if token, ok := p.match(SEMICOLON); !ok {
		p.addError(newError(token.Lexeme, []string{";"}, token.Position))
	}
	return result
}

// 	if_stmt -> IF '(' boolexpr ')' stmt ELSE stmt
func (p *Parser) IfStatement() *IfStatement {
	if _, ok := p.match(IF); !ok {
		return nil
	}
	result := &IfStatement{Position: p.lookahead.Position}

	if token, ok := p.match(LPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{"("}, token.Position))
	}
	result.Condition = p.BooleanExpression()

	if token, ok := p.match(RPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{")"}, token.Position))
	}
	result.IfBranch = p.Statement()

	if token, ok := p.match(ELSE); !ok {
		p.addError(newError(token.Lexeme, []string{"else"}, token.Position))
		return result
	}

	result.ElseBranch = p.Statement()
	return result
}

// 	while_stmt -> WHILE '(' boolexpr ')' stmt
func (p *Parser) WhileStatement() *WhileStatement {
	if _, ok := p.match(WHILE); !ok {
		return nil
	}
	result := &WhileStatement{Position: p.lookahead.Position}

	if token, ok := p.match(LPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{"("}, token.Position))
	}
	result.Condition = p.BooleanExpression()
	if token, ok := p.match(RPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{")"}, token.Position))
	}
	result.Body = p.Statement()
	return result
}

// 	switch_stmt -> SWITCH '(' expression ')' '{' caselist DEFAULT ':' stmtlist '}'
func (p *Parser) SwitchStatement() *Switch {
	if _, ok := p.match(SWITCH); !ok {
		return nil
	}
	result := &Switch{Position: p.lookahead.Position}

	if token, ok := p.match(LPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{"("}, token.Position))
	}

	if token, ok := p.match(RPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{")"}, token.Position))
	}

	if token, ok := p.match(LBRACKET); !ok {
		p.addError(newError(token.Lexeme, []string{"{"}, token.Position))
	}
	result.Cases = p.SwitchCases()

	if token, ok := p.match(DEFAULT); !ok {
		p.addError(newError(token.Lexeme, []string{"DEFAULT"}, token.Position))
	}

	if token, ok := p.match(COLON); !ok {
		p.addError(newError(token.Lexeme, []string{":"}, token.Position))
	}
	result.DefaultCase = p.Statements()

	if token, ok := p.match(RBRACKET); !ok {
		p.addError(newError(token.Lexeme, []string{"}"}, token.Position))
	}
	return result
}

func (p *Parser) SwitchCases() []SwitchCase {
	cases := []SwitchCase{}
	for p.lookahead.TokenType == CASE {
		item := SwitchCase{Position: p.lookahead.Position}
		p.match(CASE)
		if token, ok := p.match(NUM); ok {
			value, err := strconv.ParseInt(token.Lexeme, 10, 64)
			if err != nil {
				p.addError(ErrorType{Message: fmt.Sprintf("%s is not an int", token.Lexeme)})
			}
			item.Value = value
		} else {
			p.addError(newError(token.Lexeme, []string{"NUM"}, token.Position))
		}
		if token, ok := p.match(COLON); !ok {
			p.addError(newError(token.Lexeme, []string{":"}, token.Position))
		}
		item.Statements = p.Statements()
		cases = append(cases, item)
	}

	return cases
}

// 	break_stmt -> BREAK ';'
func (p *Parser) BreakStatement() *Break {
	result := &Break{Position: p.lookahead.Position}
	if _, ok := p.match(BREAK); !ok {
		return nil
	}

	// ;
	if token, ok := p.match(SEMICOLON); !ok {
		p.addError(newError(token.Lexeme, []string{";"}, token.Position))
	}

	return result
}

//	stmt_block -> '{' stmtlist '}'
func (p *Parser) StatementsBlock() *Block {
	// Parse {
	startBlock := false
	startBlockToken, startBlock := p.match(LBRACKET)
	if !startBlock {
		p.addError(newError(startBlockToken.Lexeme, []string{"{"}, startBlockToken.Position))
	}
	statements := p.Statements()
	// Only show an error for the } if there was a {
	if token, ok := p.match(RBRACKET); !ok && startBlock {
		p.addError(newError(token.Lexeme, []string{"}"}, token.Position))
	}
	return &Block{Position: startBlockToken.Position, Statements: statements}
}

//	stmtlist -> stmt stmtlist | ε
func (p *Parser) Statements() []Statement {
	statements := []Statement{}
	for {
		statement := p.Statement()
		if statement == nil {
			break
		}
		statements = append(statements, statement)
	}
	return statements
}

// 	boolexpr -> boolterm boolexpr'
// 	boolexpr' -> OR boolterm boolexpr | ε
func (p *Parser) BooleanExpression() Boolean {
	result := p.BooleanTerm()
	for p.lookahead.TokenType == OR {
		token, _ := p.match(OR)
		result = &Or{
			Position: token.Position,
			LHS:      result,
			RHS:      p.BooleanTerm(),
		}
	}

	return result
}

// 	boolterm -> boolfactor boolterm'
// 	boolterm' -> AND boolfactor boolterm' | ε
func (p *Parser) BooleanTerm() Boolean {
	result := p.BooleanFactor()
	for p.lookahead.TokenType == AND {
		token, _ := p.match(AND)
		result = &And{
			Position: token.Position,
			LHS:      result,
			RHS:      p.BooleanFactor(),
		}
	}

	return result
}

// 	boolfactor -> NOT '(' boolexpr ')' | expression RELOP expression
func (p *Parser) BooleanFactor() Boolean {
	position := p.lookahead.Position
	if p.lookahead.TokenType == NOT {
		p.match(NOT)
		if token, ok := p.match(LPAREN); !ok {
			p.addError(newError(token.Lexeme, []string{"("}, token.Position))

			expr := p.BooleanExpression()

			if token, ok := p.match(RPAREN); !ok {
				p.addError(newError(token.Lexeme, []string{")"}, token.Position))
			}

			return &Not{Position: position, Value: expr}
		}
	}
	expr := p.BooleanExpression()
	return &Not{Position: position, Value: expr}
}
