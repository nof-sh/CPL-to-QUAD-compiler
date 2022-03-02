package cpq

import (
	"fmt"
	"strconv"
	"strings"
)

// Error represents an error that occurred during code generation.
type ErrorType struct {
	Message  string
	Found    string
	Expected []string
	Pos      Position
}

// Parser represents a CPL parser.
type Parser struct {
	Errors    []ErrorType
	scanner   *Scanner
	lookahead Token
}

// Error returns the string representation of the error.
func (e *ErrorType) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s at line %d, char %d", e.Message, e.Pos.Line+1, e.Pos.Column+1)
	}
	return fmt.Sprintf("found %s, expected %s at line %d, char %d", e.Found,
		strings.Join(e.Expected, ", "), e.Pos.Line+1, e.Pos.Column+1)
}

// newParseError returns a new instance of ParseError.
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

// NewParser returns a new instance of Parser.
func NewParser(scanner *Scanner) *Parser {
	return &Parser{
		Errors:    []ErrorType{},
		scanner:   scanner,
		lookahead: scanner.Scan(),
	}
}

// Parse parses a CPL program and returns its AST representation.
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
	// Try to find the requested token.
	if token, ok := p.matchToken(tokenTypes...); ok {
		return token, true
	}

	return &p.lookahead, false
}

func (p *Parser) skip() {
	p.lookahead = p.scanner.Scan()
}

// ParseProgram parses a CPL program and returns a Program AST object.
// 	program -> declarations stmt_block
func (p *Parser) ParseProgram() *Program {
	program := &Program{Position: p.lookahead.Position}

	// Parse declarations.
	program.Declarations = p.ParseDeclarations()

	// Parse statements.
	program.StatementsBlock = p.ParseStatementsBlock()

	// Make sure there's an EOF at the end of the file.
	if token, ok := p.match(EOF); !ok {
		p.addError(newError(token.Lexeme, []string{"EOF"}, program.Position))
	}

	return program
}

// ParseDeclarations parses a list of declarations and returns a Declaration AST array.
// 	declarations -> declaration declarations | ε
func (p *Parser) ParseDeclarations() []Declaration {
	declarations := []Declaration{}
	for p.lookahead.TokenType == ID {
		declarations = append(declarations, *p.ParseDeclaration())
	}

	return declarations
}

// ParseDeclaration parses a declaration and returns a Declaration AST object.
// 	declaration -> idlist ':' type ';'
func (p *Parser) ParseDeclaration() *Declaration {
	declaration := &Declaration{Position: p.lookahead.Position}
	declaration.Names = p.ParseIDList()

	if token, ok := p.match(COLON); !ok {
		p.addError(newError(token.Lexeme, []string{":"}, token.Position))
	}

	declaration.Type = p.ParseType()
	if declaration.Type == Unknown {

	}

	if token, ok := p.match(SEMICOLON); !ok {
		p.addError(newError(token.Lexeme, []string{";"}, token.Position))
	}

	return declaration
}

// ParseType parses a type returns it as a DataType.
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

// ParseIDList parses a list of IDs and returns a string array.
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

// ParseStatement parses a CPL statement.
//	stmt -> assignment_stmt | input_stmt | output_stmt | if_stmt | while_stmt
//		| switch_stmt | break_stmt | stmt_block
func (p *Parser) ParseStatement() Statement {
	switch p.lookahead.TokenType {
	case ID:
		return p.ParseAssignmentStatement()

	case INPUT:
		return p.ParseInputStatement()

	case OUTPUT:
		return p.ParseOutputStatement()

	case IF:
		return p.ParseIfStatement()

	case WHILE:
		return p.ParseWhileStatement()

	case SWITCH:
		return p.ParseSwitchStatement()

	case BREAK:
		return p.ParseBreakStatement()

	case LBRACKET:
		return p.ParseStatementsBlock()
	}

	return nil
}

// ParseAssignmentStatement parses a CPL assignment statement.
// 	assignment_stmt -> ID '=' assignment_stmt'
// 	assignment_stmt' -> expression ';'
//   	| STATIC_CAST '(' type ')' '(' expression ')' ';
func (p *Parser) ParseAssignmentStatement() *Assignment {
	result := &Assignment{Position: p.lookahead.Position}

	// ID
	if token, ok := p.match(ID); ok {
		result.Variable = token.Lexeme
	} else {
		p.addError(newError(token.Lexeme, []string{"ID"}, token.Position))
	}

	// =
	if token, ok := p.match(EQUALS); !ok {
		p.addError(newError(token.Lexeme, []string{"ID"}, token.Position))
	}

	// Parse static_cast(type) if exists
	if p.lookahead.TokenType == STATICCAST {
		p.match(STATICCAST)

		// (
		if token, ok := p.match(LPAREN); !ok {
			p.addError(newError(token.Lexeme, []string{"("}, token.Position))
		}

		result.CastType = p.ParseType()

		// )
		if token, ok := p.match(RPAREN); !ok {
			p.addError(newError(token.Lexeme, []string{")"}, token.Position))
		}
	}

	// ;
	if token, ok := p.match(SEMICOLON); !ok {
		p.addError(newError(token.Lexeme, []string{";"}, token.Position))
	}

	return result
}

// ParseInputStatement parses a CPL input statement, which can be used for retrieving
// user input.
// 	input_stmt -> INPUT '(' ID ')' ';'
func (p *Parser) ParseInputStatement() *Input {
	if _, ok := p.match(INPUT); !ok {
		return nil
	}

	result := &Input{Position: p.lookahead.Position}

	// (
	if token, ok := p.match(LPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{"("}, token.Position))
	}

	// ID
	if token, ok := p.match(ID); ok {
		result.Variable = token.Lexeme
	} else {
		p.addError(newError(token.Lexeme, []string{"ID"}, token.Position))
	}

	// )
	if token, ok := p.match(RPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{")"}, token.Position))
	}

	// ;
	if token, ok := p.match(SEMICOLON); !ok {
		p.addError(newError(token.Lexeme, []string{";"}, token.Position))
	}

	return result
}

// ParseOutputStatement parses a CPL output statement, which can be used for printing
// expressions.
// 	output_stmt -> OUTPUT '(' expression ')' ';'
func (p *Parser) ParseOutputStatement() *Output {
	if _, ok := p.match(OUTPUT); !ok {
		return nil
	}

	result := &Output{Position: p.lookahead.Position}

	// (
	if token, ok := p.match(LPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{"("}, token.Position))
	}

	// )
	if token, ok := p.match(RPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{")"}, token.Position))
	}

	// ;
	if token, ok := p.match(SEMICOLON); !ok {
		p.addError(newError(token.Lexeme, []string{";"}, token.Position))
	}

	return result
}

// ParseIfStatement parses a CPL if statement.
// 	if_stmt -> IF '(' boolexpr ')' stmt ELSE stmt
func (p *Parser) ParseIfStatement() *IfStatement {
	if _, ok := p.match(IF); !ok {
		return nil
	}

	result := &IfStatement{Position: p.lookahead.Position}

	// (
	if token, ok := p.match(LPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{"("}, token.Position))
	}

	result.Condition = p.ParseBooleanExpression()

	// )
	if token, ok := p.match(RPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{")"}, token.Position))
	}

	// stmt
	result.IfBranch = p.ParseStatement()

	// ELSE
	if token, ok := p.match(ELSE); !ok {
		p.addError(newError(token.Lexeme, []string{"else"}, token.Position))
		return result
	}

	// stmt
	result.ElseBranch = p.ParseStatement()

	return result
}

// ParseWhileStatement parses a CPL if statement.
// 	while_stmt -> WHILE '(' boolexpr ')' stmt
func (p *Parser) ParseWhileStatement() *WhileStatement {
	if _, ok := p.match(WHILE); !ok {
		return nil
	}

	result := &WhileStatement{Position: p.lookahead.Position}

	// (
	if token, ok := p.match(LPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{"("}, token.Position))
	}

	result.Condition = p.ParseBooleanExpression()

	// )
	if token, ok := p.match(RPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{")"}, token.Position))
	}

	// stmt
	result.Body = p.ParseStatement()
	return result
}

// ParseSwitchStatement parses a CPL switch statement.
// 	switch_stmt -> SWITCH '(' expression ')' '{' caselist DEFAULT ':' stmtlist '}'
func (p *Parser) ParseSwitchStatement() *Switch {
	if _, ok := p.match(SWITCH); !ok {
		return nil
	}

	result := &Switch{Position: p.lookahead.Position}

	// (
	if token, ok := p.match(LPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{"("}, token.Position))
	}

	// )
	if token, ok := p.match(RPAREN); !ok {
		p.addError(newError(token.Lexeme, []string{")"}, token.Position))
	}

	// {
	if token, ok := p.match(LBRACKET); !ok {
		p.addError(newError(token.Lexeme, []string{"{"}, token.Position))
	}

	result.Cases = p.ParseSwitchCases()

	// DEFAULT
	if token, ok := p.match(DEFAULT); !ok {
		p.addError(newError(token.Lexeme, []string{"DEFAULT"}, token.Position))
	}

	// :
	if token, ok := p.match(COLON); !ok {
		p.addError(newError(token.Lexeme, []string{":"}, token.Position))
	}

	result.DefaultCase = p.ParseStatements()

	// }
	if token, ok := p.match(RBRACKET); !ok {
		p.addError(newError(token.Lexeme, []string{"}"}, token.Position))
	}

	return result
}

// ParseSwitchCases parses zero or more switch cases.
//	CASE NUM ':' stmtlist caselist
func (p *Parser) ParseSwitchCases() []SwitchCase {
	cases := []SwitchCase{}
	for p.lookahead.TokenType == CASE {
		item := SwitchCase{Position: p.lookahead.Position}
		p.match(CASE)

		// NUM
		if token, ok := p.match(NUM); ok {
			value, err := strconv.ParseInt(token.Lexeme, 10, 64)
			if err != nil {
				p.addError(ErrorType{Message: fmt.Sprintf("%s is not an int", token.Lexeme)})
			}

			item.Value = value
		} else {
			p.addError(newError(token.Lexeme, []string{"NUM"}, token.Position))
		}

		// :
		if token, ok := p.match(COLON); !ok {
			p.addError(newError(token.Lexeme, []string{":"}, token.Position))
		}

		item.Statements = p.ParseStatements()

		cases = append(cases, item)
	}

	return cases
}

// ParseBreakStatement parses a CPL break statement.
// 	break_stmt -> BREAK ';'
func (p *Parser) ParseBreakStatement() *Break {
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

// ParseStatementsBlock parses a block of statements.
//	stmt_block -> '{' stmtlist '}'
func (p *Parser) ParseStatementsBlock() *Block {
	// Parse {
	startBlock := false
	startBlockToken, startBlock := p.match(LBRACKET)
	if !startBlock {
		p.addError(newError(startBlockToken.Lexeme, []string{"{"}, startBlockToken.Position))
	}

	statements := p.ParseStatements()

	// Parse }
	// Only show an error for the } if there was a {
	if token, ok := p.match(RBRACKET); !ok && startBlock {
		p.addError(newError(token.Lexeme, []string{"}"}, token.Position))
	}

	return &Block{Position: startBlockToken.Position, Statements: statements}
}

// ParseStatements parses zero or more statements.
//	stmtlist -> stmt stmtlist | ε
func (p *Parser) ParseStatements() []Statement {
	statements := []Statement{}
	for {
		statement := p.ParseStatement()
		if statement == nil {
			break
		}

		statements = append(statements, statement)
	}

	return statements
}

// ParseBooleanExpression parses expressions that might contain any boolean operator.
// 	boolexpr -> boolterm boolexpr'
// 	boolexpr' -> OR boolterm boolexpr | ε
func (p *Parser) ParseBooleanExpression() Boolean {
	result := p.ParseBooleanTerm()
	for p.lookahead.TokenType == OR {
		token, _ := p.match(OR)
		result = &Or{
			Position: token.Position,
			LHS:      result,
			RHS:      p.ParseBooleanTerm(),
		}
	}

	return result
}

// ParseBooleanTerm parses expressions that might contain AND operator.
// 	boolterm -> boolfactor boolterm'
// 	boolterm' -> AND boolfactor boolterm' | ε
func (p *Parser) ParseBooleanTerm() Boolean {
	result := p.ParseBooleanFactor()
	for p.lookahead.TokenType == AND {
		token, _ := p.match(AND)
		result = &And{
			Position: token.Position,
			LHS:      result,
			RHS:      p.ParseBooleanFactor(),
		}
	}

	return result
}

// ParseBooleanFactor parses a boolean expression with NOT operator or a relational operator.
// 	boolfactor -> NOT '(' boolexpr ')'
//		| expression RELOP expression
func (p *Parser) ParseBooleanFactor() Boolean {
	position := p.lookahead.Position
	if p.lookahead.TokenType == NOT {
		p.match(NOT)
		if token, ok := p.match(LPAREN); !ok {
			p.addError(newError(token.Lexeme, []string{"("}, token.Position))

			expr := p.ParseBooleanExpression()

			if token, ok := p.match(RPAREN); !ok {
				p.addError(newError(token.Lexeme, []string{")"}, token.Position))
			}

			return &Not{Position: position, Value: expr}
		}
	}
	expr := p.ParseBooleanExpression()
	return &Not{Position: position, Value: expr}
}
