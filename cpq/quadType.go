package cpq

// DataType represents the primitive data types available in CPL.
type DataType int

const (
	// Unknown primitive data type.
	Unknown DataType = iota
	// Float means the data type is a float.
	Float DataType = 1
	// Integer means the data type is an integer.
	Integer DataType = 2
)

// Operator represents a boolean or arithmatic operator in CPL.
type Operator int

// Types of operators
const (
	Add                  Operator = iota // +
	Subtract                             // -
	Multiply                             // *
	Divide                               // /
	EqualTo                              // ==
	NotEqualTo                           // !=
	GreaterThan                          // >
	LessThan                             // <
	GreaterThanOrEqualTo                 // >=
	LessThenOrEqualTo                    // <=
)

// Node represents a node in the CPL abstract syntax tree.
type Node interface {
	// node is unexported to ensure implementations of Node
	// can only originate in this package.
	node()
}

// Expression is a combination of numbers, variables and operators that
// can be evaluated to a value.
type NodeExpression interface {
	Node
	// expression is unexported to ensure implementations of Expression
	// can only originate in this package.
	expression()
}

// BooleanExpression can be evaulated to a boolean value (true or false).
// NOTE: In CPL, BooleanExpression isn't an Expression! These are two distinct types.
type NodeBoolean interface {
	Node
	// boolexpr is unexported to ensure implementations of Expression
	// can only originate in this package.
	boolexpr()
}

// Program represents the root node of a CPL program.
type Program struct {
	Declarations    []Declaration
	StatementsBlock *Block
	Position        Position
}

// Declaration of one or more variables.
type Declaration struct {
	Names    []string
	Type     DataType
	Position Position
}

// Statement represents a single command in CPL.
type Statement interface {
	Node
	// statement is unexported to ensure implementations of Statement
	// can only originate in this package.
	statement()
}

// AssignmentStatement represents a command for assigning a value to a variable,
// e.g: x = 5;
type Assignment struct {
	Variable string
	Value    Expression
	// If the assignment doesn't contain static_cast<>, then CastType will be Unknown.
	// Otherwise, CastType will contain the type to cast to.
	CastType DataType
	Position Position
}

// InputStatement represents a command for retrieving user input to a variable.
// e.g: input(a);
type Input struct {
	Variable string
	Position Position
}

// OutputStatement represents a command for printing an expression.
// e.g: output(x + y);
type Output struct {
	Value    Expression
	Position Position
}

// IfStatement represents a conditional command. In CPL, if statements must contain an else clause!
// e.g: if (x == y) { output(x); } else { output(y); }
type IfStatement struct {
	Condition  Boolean
	IfBranch   Statement
	ElseBranch Statement
	Position   Position
}

// WhileStatement is a control flow statement that allows code to be executed
// repeatedly based on a given Boolean condition.
type WhileStatement struct {
	Condition Boolean
	Body      Statement
	Position  Position
}

// SwitchStatement is a type of selection control mechanism used to allow the value of
// a variable or expression to change the control flow of program execution.
type Switch struct {
	Expression  Expression
	Cases       []SwitchCase
	DefaultCase []Statement
	Position    Position
}

// SwitchCase represents a flow for a specific value in a switch statement.
type SwitchCase struct {
	Value      int64
	Statements []Statement
	Position   Position
}

// BreakStatement represents a statement that exits from a switch case
// or a while loop.
type Break struct {
	Position Position
}

// StatementsBlock represents a block of sentences, e.g { s1; s2; s3; }.
// It is itself a statement.
type Block struct {
	Statements []Statement
	Position   Position
}

// BooleanExpression can be evaulated to a boolean value (true or false).
// NOTE: In CPL, BooleanExpression isn't an Expression! These are two distinct types.
type Boolean interface {
	Node
	// boolexpr is unexported to ensure implementations of Expression
	// can only originate in this package.
	boolexpr()
}

// VariableExpression is an expression that contains a single variable.
type Variable struct {
	Variable string
	Position Position
}

// IntLiteral is an expression that contains a single constant integer number.
type IntNum struct {
	Value    int64
	Position Position
}

// FloatLiteral is an expression that contains a single constant integer number.
type FloatNum struct {
	Value    float64
	Position Position
}

// ArithmeticExpression is an expression that contains a +, -, *, / operator.
type Arithmetic struct {
	LHS      Expression
	Operator Operator
	RHS      Expression
	Position Position
}

// OrBooleanExpression is a boolean expression that has an OR operator.
type Or struct {
	LHS      Boolean
	RHS      Boolean
	Position Position
}

// AndBooleanExpression is a boolean expression that has an AND operator.
type And struct {
	LHS      Boolean
	RHS      Boolean
	Position Position
}

// NotBooleanExpression is a boolean expression that has a NOT operator.
type Not struct {
	Value    Boolean
	Position Position
}

// CompareBooleanExpression is a boolean expression that compares between two expressions,
// e.g x < y
type Compare struct {
	LHS      Expression
	Operator Operator
	RHS      Expression
	Position Position
}

func (*Program) node()        {}
func (*Declaration) node()    {}
func (*Assignment) node()     {}
func (*Input) node()          {}
func (*Output) node()         {}
func (*IfStatement) node()    {}
func (*WhileStatement) node() {}
func (*Switch) node()         {}
func (*SwitchCase) node()     {}
func (*Break) node()          {}
func (*Block) node()          {}
func (*Variable) node()       {}
func (*IntNum) node()         {}
func (*FloatNum) node()       {}
func (*Arithmetic) node()     {}
func (*Or) node()             {}
func (*And) node()            {}
func (*Not) node()            {}
func (*Compare) node()        {}

func (*Assignment) statement()     {}
func (*Input) statement()          {}
func (*Output) statement()         {}
func (*IfStatement) statement()    {}
func (*WhileStatement) statement() {}
func (*Switch) statement()         {}
func (*Break) statement()          {}
func (*Block) statement()          {}

func (*Variable) expression()   {}
func (*IntNum) expression()     {}
func (*FloatNum) expression()   {}
func (*Arithmetic) expression() {}

func (*Or) boolexpr()      {}
func (*And) boolexpr()     {}
func (*Not) boolexpr()     {}
func (*Compare) boolexpr() {}
