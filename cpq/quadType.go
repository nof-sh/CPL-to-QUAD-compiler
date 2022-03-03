package cpq

// Data type in CPL.
type DataType int

const (
	Unknown DataType = iota
	Float   DataType = 1
	Integer DataType = 2
)

//operator in CPL.
type Operator int

//operators
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

type Node interface {
	node()
}

type NodeExpression interface {
	Node
	expression()
}

type NodeBoolean interface {
	Node
	boolexpr()
}

// a CPL program.
type Program struct {
	Declarations    []Declaration
	StatementsBlock *Block
	Pos             Position
}

type Declaration struct {
	Names []string
	Type  DataType
	Pos   Position
}

type Statement interface {
	Node
	statement()
}

type Assignment struct {
	Variable string
	Val      Expression
	CastType DataType
	Pos      Position
}

type Input struct {
	Variable string
	Pos      Position
}

type Output struct {
	Value    Expression
	Position Position
}

type IfStatement struct {
	Condition  Boolean
	IfBranch   Statement
	ElseBranch Statement
	Position   Position
}

type WhileStatement struct {
	Condition Boolean
	Body      Statement
	Position  Position
}

type Switch struct {
	Expression  Expression
	Cases       []SwitchCase
	DefaultCase []Statement
	Position    Position
}

type SwitchCase struct {
	Value      int64
	Statements []Statement
	Position   Position
}

type Break struct {
	Position Position
}

type Block struct {
	Statements []Statement
	Position   Position
}

type Boolean interface {
	Node
	boolexpr()
}

type Variable struct {
	Variable string
	Position Position
}

type IntNum struct {
	Value    int64
	Position Position
}

type FloatNum struct {
	Value    float64
	Position Position
}

type Arithmetic struct {
	LHS      Expression
	Operator Operator
	RHS      Expression
	Position Position
}

type Or struct {
	LHS      Boolean
	RHS      Boolean
	Position Position
}

type And struct {
	LHS      Boolean
	RHS      Boolean
	Position Position
}

type Not struct {
	Value    Boolean
	Position Position
}

type Compare struct {
	LHS      Expression
	Operator Operator
	RHS      Expression
	Position Position
}

func (*Program) node()             {}
func (*Declaration) node()         {}
func (*Assignment) node()          {}
func (*Input) node()               {}
func (*Output) node()              {}
func (*IfStatement) node()         {}
func (*WhileStatement) node()      {}
func (*Switch) node()              {}
func (*SwitchCase) node()          {}
func (*Break) node()               {}
func (*Block) node()               {}
func (*Variable) node()            {}
func (*IntNum) node()              {}
func (*FloatNum) node()            {}
func (*Arithmetic) node()          {}
func (*Or) node()                  {}
func (*And) node()                 {}
func (*Not) node()                 {}
func (*Compare) node()             {}
func (*Assignment) statement()     {}
func (*Input) statement()          {}
func (*Output) statement()         {}
func (*IfStatement) statement()    {}
func (*WhileStatement) statement() {}
func (*Switch) statement()         {}
func (*Break) statement()          {}
func (*Block) statement()          {}
func (*Variable) expression()      {}
func (*IntNum) expression()        {}
func (*FloatNum) expression()      {}
func (*Arithmetic) expression()    {}
func (*Or) boolexpr()              {}
func (*And) boolexpr()             {}
func (*Not) boolexpr()             {}
func (*Compare) boolexpr()         {}
