package cpq

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type CodeGenerator struct {
	Errors         []Error
	output         *bufio.Writer
	Variables      map[string]DataType
	temporaryIndex int
	labelIndex     int
	breakStack     []string
}

type Expression struct {
	Code string
	Type DataType
}

// NewCodeGenerator returns a new instance of CodeGenerator.
func NewCodeGenerator(output io.Writer) *CodeGenerator {
	return &CodeGenerator{
		Errors:         []Error{},
		output:         bufio.NewWriterSize(output, 1),
		Variables:      map[string]parser.DataType{},
		temporaryIndex: 0,
		labelIndex:     0,
		breakStack:     []string{},
	}
}

// Codegen generates code to an output file
func Codegen(program *parser.Program) (string, []Error) {
	buf := new(bytes.Buffer)

	c := NewCodeGenerator(buf)
	c.CodegenProgram(program)

	return buf.String(), c.Errors
}

// CodegenProgram generates code for a CPL program.
func (c *CodeGenerator) CodegenProgram(node *parser.Program) {
	// Go over variable declarations
	for _, declaration := range node.Declarations {
		for _, name := range declaration.Names {
			if _, exists := c.Variables[name]; exists {
				c.Errors = append(c.Errors, Error{
					Message: fmt.Sprintf("variable %s already defined", name),
					Pos:     declaration.Position,
				})
				continue
			}

			c.Variables[name] = declaration.Type
		}
	}

	c.CodegenStatement(node.StatementsBlock)
	c.output.WriteString("HALT\n")
}

// CodegenStatement generates code for a CPL statement.
func (c *CodeGenerator) CodegenStatement(node parser.Statement) {
	switch s := node.(type) {
	case *parser.AssignmentStatement:
		c.CodegenAssignmentStatement(s)
	case *parser.InputStatement:
		c.CodegenInputStatement(s)
	case *parser.OutputStatement:
		c.CodegenOutputStatement(s)
	case *parser.IfStatement:
		c.CodegenIfStatement(s)
	case *parser.WhileStatement:
		c.CodegenWhileStatement(s)
	case *parser.SwitchStatement:
		c.CodegenSwitchStatement(s)
	case *parser.BreakStatement:
		c.CodegenBreakStatement(s)
	case *parser.StatementsBlock:
		c.CodegenStatementsBlock(s)
	}
}

// CodegenAssignmentStatement generates code for assignment statements.
func (c *CodeGenerator) CodegenAssignmentStatement(node *parser.AssignmentStatement) {
	exp := c.CodegenExpression(node.Value)

	// Make sure the variable is defined.
	if _, exists := c.Variables[node.Variable]; !exists {
		c.Errors = append(c.Errors, Error{
			Message: fmt.Sprintf("undefined variable %s", node.Variable),
			Pos:     node.Position,
		})
		return
	}

	if exp == nil {
		return
	}

	// Cast type if there's a static_cast
	if node.CastType != parser.Unknown && node.CastType != exp.Type {
		exp = c.codegenCastExpression(exp, node.CastType)
	}

	// Make sure the expression's type is okay
	if c.Variables[node.Variable] == parser.Integer && exp.Type == parser.Float {
		c.Errors = append(c.Errors, Error{
			Message: fmt.Sprintf("cannot assign float value to int variable %s", node.Variable),
			Pos:     node.Position,
		})
		return
	}

	// If the variable is float but the expression is integer, cast it to float.
	if c.Variables[node.Variable] == parser.Float && exp.Type == parser.Integer {
		exp = c.codegenCastExpression(exp, parser.Float)
	}

	// Codegen
	if c.Variables[node.Variable] == parser.Integer {
		c.output.WriteString(fmt.Sprintf("IASN %s %s\n", node.Variable, exp.Code))
	} else if c.Variables[node.Variable] == parser.Float {
		c.output.WriteString(fmt.Sprintf("RASN %s %s\n", node.Variable, exp.Code))
	}
}

// CodegenInputStatement generates code for input statements.
func (c *CodeGenerator) CodegenInputStatement(node *parser.InputStatement) {
	// Make sure the variable is defined.
	if _, exists := c.Variables[node.Variable]; !exists {
		c.Errors = append(c.Errors, Error{
			Message: fmt.Sprintf("undefined variable %s", node.Variable),
			Pos:     node.Position,
		})
		return
	}

	if c.Variables[node.Variable] == parser.Integer {
		c.output.WriteString(fmt.Sprintf("IINP %s\n", node.Variable))
	} else if c.Variables[node.Variable] == parser.Float {
		c.output.WriteString(fmt.Sprintf("RINP %s\n", node.Variable))
	}
}

// CodegenOutputStatement generates code for output statements.
func (c *CodeGenerator) CodegenOutputStatement(node *parser.OutputStatement) {
	exp := c.CodegenExpression(node.Value)
	if exp == nil {
		return
	}

	if exp.Type == parser.Integer {
		c.output.WriteString(fmt.Sprintf("IPRT %s\n", exp.Code))
	} else if exp.Type == parser.Float {
		c.output.WriteString(fmt.Sprintf("RPRT %s\n", exp.Code))
	}
}

// CodegenIfStatement generates code for if statements.
func (c *CodeGenerator) CodegenIfStatement(node *parser.IfStatement) {
	condition := c.CodegenBooleanExpression(node.Condition)
	endIfLabel := c.getNewLabel()

	// Even though in CPL you can't write an if statement without an else, we still want
	// to support that because switch statements, which are implemented through if statements,
	// don't need else.
	var elseLabel string
	if node.ElseBranch != nil {
		elseLabel = c.getNewLabel()
		c.output.WriteString(fmt.Sprintf("JMPZ %s %s\n", elseLabel, condition))
	} else {
		c.output.WriteString(fmt.Sprintf("JMPZ %s %s\n", endIfLabel, condition))
	}

	c.CodegenStatement(node.IfBranch)

	if node.ElseBranch != nil {
		c.output.WriteString(fmt.Sprintf("JUMP %s\n", endIfLabel))
		c.output.WriteString(fmt.Sprintf("%s:\n", elseLabel))
		c.CodegenStatement(node.ElseBranch)
	}

	c.output.WriteString(fmt.Sprintf("%s:\n", endIfLabel))
}

// CodegenWhileStatement generates code for while statements.
func (c *CodeGenerator) CodegenWhileStatement(node *parser.WhileStatement) {
	conditionLabel := c.getNewLabel()
	endLoopLabel := c.getNewLabel()

	c.output.WriteString(fmt.Sprintf("%s:\n", conditionLabel))
	condition := c.CodegenBooleanExpression(node.Condition)
	c.output.WriteString(fmt.Sprintf("JMPZ %s %s\n", endLoopLabel, condition))

	c.breakStack = append(c.breakStack, endLoopLabel)
	c.CodegenStatement(node.Body)
	if c.breakStack[len(c.breakStack)-1] == endLoopLabel {
		c.breakStack = c.breakStack[:len(c.breakStack)-1]
	}

	c.output.WriteString(fmt.Sprintf("JUMP %s\n", conditionLabel))
	c.output.WriteString(fmt.Sprintf("%s:\n", endLoopLabel))
}

// CodegenSwitchStatement generates code for switch statements.
func (c *CodeGenerator) CodegenSwitchStatement(node *parser.SwitchStatement) {
	// Evaluate expression
	exp := c.CodegenExpression(node.Expression)
	if exp == nil {
		return
	}

	if exp.Type != parser.Integer {
		c.Errors = append(c.Errors, Error{
			Message: fmt.Sprintf("switch expression must be an integer"),
			Pos:     node.Position,
		})
	}

	temp := c.getNewTemporary()
	caseLabels := map[int]string{}

	// Generate if statement for each case
	for i, switchCase := range node.Cases {
		caseLabels[i] = c.getNewLabel()
		c.output.WriteString(fmt.Sprintf("INQL %s %s %d\n", temp, exp.Code, switchCase.Value))
		c.output.WriteString(fmt.Sprintf("JMPZ %s %s\n", caseLabels[i], temp))
	}

	defaultLabel := c.getNewLabel()
	endSwitchLabel := c.getNewLabel()
	c.output.WriteString(fmt.Sprintf("JUMP %s\n", defaultLabel))

	c.breakStack = append(c.breakStack, endSwitchLabel)

	// Generate labels and code for each case
	for i, switchCase := range node.Cases {
		c.output.WriteString(fmt.Sprintf("%s:\n", caseLabels[i]))
		c.CodegenStatement(&parser.StatementsBlock{
			Statements: switchCase.Statements,
		})
	}

	// Default case
	c.output.WriteString(fmt.Sprintf("%s:\n", defaultLabel))
	c.CodegenStatement(&parser.StatementsBlock{
		Statements: node.DefaultCase,
	})

	if c.breakStack[len(c.breakStack)-1] == endSwitchLabel {
		c.breakStack = c.breakStack[:len(c.breakStack)-1]
	}

	c.output.WriteString(fmt.Sprintf("%s:\n", endSwitchLabel))
}

// CodegenBreakStatement generates code for break statements.
func (c *CodeGenerator) CodegenBreakStatement(node *parser.BreakStatement) {
	if len(c.breakStack) == 0 {
		c.Errors = append(c.Errors, Error{
			Message: fmt.Sprintf("break statement must be inside a while loop or a switch case"),
			Pos:     node.Position,
		})
		return
	}

	c.output.WriteString(fmt.Sprintf("JUMP %s\n", c.breakStack[len(c.breakStack)-1]))
}

// CodegenStatementsBlock generates code for a statements block.
func (c *CodeGenerator) CodegenStatementsBlock(node *parser.StatementsBlock) {
	for _, statement := range node.Statements {
		c.CodegenStatement(statement)
	}
}

// CodegenExpression generates code for a CPL expression.
func (c *CodeGenerator) CodegenExpression(node parser.Expression) *Expression {
	switch s := node.(type) {
	case *parser.ArithmeticExpression:
		return c.CodegenArithmeticExpression(s)
	case *parser.VariableExpression:
		return c.CodegenVariableExpression(s)
	case *parser.IntLiteral:
		return c.CodegenIntLiteral(s)
	case *parser.FloatLiteral:
		return c.CodegenFloatLiteral(s)
	}

	return nil
}

// CodegenArithmeticExpression generates code for an arithmetic expression.
func (c *CodeGenerator) CodegenArithmeticExpression(node *parser.ArithmeticExpression) *Expression {
	lhs := c.CodegenExpression(node.LHS)
	rhs := c.CodegenExpression(node.RHS)
	if lhs == nil || rhs == nil {
		return nil
	}

	result := &Expression{
		Code: c.getNewTemporary(),
		Type: calculateExpressionType(lhs.Type, rhs.Type),
	}

	// Cast integer values to float if necessary
	if result.Type == parser.Float {
		lhs = c.codegenCastExpression(lhs, parser.Float)
		rhs = c.codegenCastExpression(rhs, parser.Float)
	}

	switch node.Operator {
	case parser.Add:
		if result.Type == parser.Integer {
			c.output.WriteString(fmt.Sprintf("IADD %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		} else if result.Type == parser.Float {
			c.output.WriteString(fmt.Sprintf("RADD %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		}

	case parser.Subtract:
		if result.Type == parser.Integer {
			c.output.WriteString(fmt.Sprintf("ISUB %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		} else if result.Type == parser.Float {
			c.output.WriteString(fmt.Sprintf("RSUB %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		}

	case parser.Multiply:
		if result.Type == parser.Integer {
			c.output.WriteString(fmt.Sprintf("IMLT %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		} else if result.Type == parser.Float {
			c.output.WriteString(fmt.Sprintf("RMLT %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		}

	case parser.Divide:
		if result.Type == parser.Integer {
			c.output.WriteString(fmt.Sprintf("IDIV %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		} else if result.Type == parser.Float {
			c.output.WriteString(fmt.Sprintf("RDIV %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		}
	}

	return result
}

// CodegenVariableExpression generates code for a variable expression.
func (c *CodeGenerator) CodegenVariableExpression(node *parser.VariableExpression) *Expression {
	// Make sure the variable is defined.
	if _, exists := c.Variables[node.Variable]; !exists {
		c.Errors = append(c.Errors, Error{
			Message: fmt.Sprintf("undefined variable %s", node.Variable),
			Pos:     node.Position,
		})
		return nil
	}

	return &Expression{Code: node.Variable, Type: c.Variables[node.Variable]}
}

// CodegenIntLiteral generates code for an integer literal.
func (c *CodeGenerator) CodegenIntLiteral(node *parser.IntLiteral) *Expression {
	return &Expression{
		Code: fmt.Sprintf("%d", node.Value),
		Type: parser.Integer,
	}
}

// CodegenFloatLiteral generates code for an float literal.
func (c *CodeGenerator) CodegenFloatLiteral(node *parser.FloatLiteral) *Expression {
	return &Expression{
		Code: fmt.Sprintf("%f", node.Value),
		Type: parser.Float,
	}
}

// CodegenBooleanExpression generates code for a CPL boolean expression, and returns
// the temporary variable that stores its result.
func (c *CodeGenerator) CodegenBooleanExpression(node parser.BooleanExpression) string {
	switch s := node.(type) {
	case *parser.OrBooleanExpression:
		return c.CodegenOrBooleanExpression(s)
	case *parser.AndBooleanExpression:
		return c.CodegenAndBooleanExpression(s)
	case *parser.NotBooleanExpression:
		return c.CodegenNotBooleanExpression(s)
	case *parser.CompareBooleanExpression:
		return c.CodegenCompareBooleanExpression(s)
	}

	return ""
}

// CodegenOrBooleanExpression generates code for a boolean OR operation.
func (c *CodeGenerator) CodegenOrBooleanExpression(node *parser.OrBooleanExpression) string {
	lhs := c.CodegenBooleanExpression(node.LHS)
	rhs := c.CodegenBooleanExpression(node.RHS)
	if lhs == "" || rhs == "" {
		return ""
	}

	result := c.getNewTemporary()

	// After the following operation:
	//   lhs=0 and rhs=0 => result will contain 0+0=0.
	//   lhs=1 and rhs=0 => result will contain 1+0=1.
	//   lhs=0 and rhs=1 => result will contain 0+1=1.
	//   lhs=1 and rhs=1 => result will contain 1+1=2.
	c.output.WriteString(fmt.Sprintf("IADD %s %s %s\n", result, lhs, rhs))

	// If result > 0 (which is always the case unless lhs=rhs=0), make it 1.
	// This is necessary because if lhs=rhs=1, then result is 2 which is an illegal boolean value.
	c.output.WriteString(fmt.Sprintf("IGRT %s %s 0\n", result, result))

	return result
}

// CodegenAndBooleanExpression generates code for a boolean AND operation.
func (c *CodeGenerator) CodegenAndBooleanExpression(node *parser.AndBooleanExpression) string {
	lhs := c.CodegenBooleanExpression(node.LHS)
	rhs := c.CodegenBooleanExpression(node.RHS)
	if lhs == "" || rhs == "" {
		return ""
	}

	result := c.getNewTemporary()

	// After the following operation:
	//   lhs=0 and rhs=0 => result will contain 0*0=0.
	//   lhs=1 and rhs=0 => result will contain 1*0=0.
	//   lhs=0 and rhs=1 => result will contain 0*1=0.
	//   lhs=1 and rhs=1 => result will contain 1*1=1.
	c.output.WriteString(fmt.Sprintf("IMLT %s %s %s\n", result, lhs, rhs))

	return result
}

// CodegenNotBooleanExpression generates code for a boolean NOT operation.
func (c *CodeGenerator) CodegenNotBooleanExpression(node *parser.NotBooleanExpression) string {
	value := c.CodegenBooleanExpression(node.Value)
	if value == "" {
		return ""
	}

	result := c.getNewTemporary()

	// After the following operation:
	//   value=0 => result will contain 1-0=1.
	//   value=1 => result will contain 1-1=0.
	c.output.WriteString(fmt.Sprintf("ISUB %s 1 %s\n", result, value))

	return result
}

// CodegenCompareBooleanExpression generates code for a expression comparison.
func (c *CodeGenerator) CodegenCompareBooleanExpression(node *parser.CompareBooleanExpression) string {
	// If the operator is x >= y, convert the AST to x == y || x > y
	if node.Operator == parser.GreaterThanOrEqualTo {
		return c.CodegenOrBooleanExpression(&parser.OrBooleanExpression{
			LHS: &parser.CompareBooleanExpression{
				LHS:      node.LHS,
				Operator: parser.EqualTo,
				RHS:      node.RHS,
			},
			RHS: &parser.CompareBooleanExpression{
				LHS:      node.LHS,
				Operator: parser.GreaterThan,
				RHS:      node.RHS,
			},
		})
	}

	// If the operator is x <= y, convert the AST to x == y || x < y
	if node.Operator == parser.LessThenOrEqualTo {
		return c.CodegenOrBooleanExpression(&parser.OrBooleanExpression{
			LHS: &parser.CompareBooleanExpression{
				LHS:      node.LHS,
				Operator: parser.EqualTo,
				RHS:      node.RHS,
			},
			RHS: &parser.CompareBooleanExpression{
				LHS:      node.LHS,
				Operator: parser.LessThan,
				RHS:      node.RHS,
			},
		})
	}

	lhs := c.CodegenExpression(node.LHS)
	rhs := c.CodegenExpression(node.RHS)
	if lhs == nil || rhs == nil {
		return ""
	}

	// Calculate the type for the expression comparison
	compareType := calculateExpressionType(lhs.Type, rhs.Type)

	// If the comparison is on floats but one of the operands are integers, cast them to floats.
	if compareType == parser.Float {
		lhs = c.codegenCastExpression(lhs, parser.Float)
		rhs = c.codegenCastExpression(rhs, parser.Float)
	}

	result := c.getNewTemporary()

	switch node.Operator {
	case parser.EqualTo:
		if compareType == parser.Integer {
			c.output.WriteString(fmt.Sprintf("IEQL %s %s %s\n", result, lhs.Code, rhs.Code))
		} else if compareType == parser.Float {
			c.output.WriteString(fmt.Sprintf("REQL %s %s %s\n", result, lhs.Code, rhs.Code))
		}

	case parser.NotEqualTo:
		if compareType == parser.Integer {
			c.output.WriteString(fmt.Sprintf("INQL %s %s %s\n", result, lhs.Code, rhs.Code))
		} else if compareType == parser.Float {
			c.output.WriteString(fmt.Sprintf("RNQL %s %s %s\n", result, lhs.Code, rhs.Code))
		}

	case parser.GreaterThan:
		if compareType == parser.Integer {
			c.output.WriteString(fmt.Sprintf("IGRT %s %s %s\n", result, lhs.Code, rhs.Code))
		} else if compareType == parser.Float {
			c.output.WriteString(fmt.Sprintf("RGRT %s %s %s\n", result, lhs.Code, rhs.Code))
		}

	case parser.LessThan:
		if compareType == parser.Integer {
			c.output.WriteString(fmt.Sprintf("ILSS %s %s %s\n", result, lhs.Code, rhs.Code))
		} else if compareType == parser.Float {
			c.output.WriteString(fmt.Sprintf("RLSS %s %s %s\n", result, lhs.Code, rhs.Code))
		}
	}

	return result
}

func (c *CodeGenerator) getNewTemporary() string {
	c.temporaryIndex++
	return fmt.Sprintf("_t%d", c.temporaryIndex)
}

func (c *CodeGenerator) getNewLabel() string {
	c.labelIndex++
	return fmt.Sprintf("@%d", c.labelIndex)
}

func (c *CodeGenerator) codegenCastExpression(exp *Expression, targetType parser.DataType) *Expression {
	if exp.Type == targetType {
		return exp
	}

	result := &Expression{
		Code: c.getNewTemporary(),
		Type: targetType,
	}

	switch targetType {
	case parser.Integer:
		c.output.WriteString(fmt.Sprintf("RTOI %s %s\n", result.Code, exp.Code))
	case parser.Float:
		c.output.WriteString(fmt.Sprintf("ITOR %s %s\n", result.Code, exp.Code))
	default:
		panic("Invalid type!")
	}

	return result
}

func calculateExpressionType(types ...parser.DataType) parser.DataType {
	for _, t := range types {
		if t == parser.Float {
			return parser.Float
		}
	}

	return parser.Integer
}

// RemoveLabels removes any labels generated by this module.
func RemoveLabels(quad string) string {
	labels := 0
	for i, line := range strings.Split(quad, "\n") {
		if strings.HasSuffix(line, ":") {
			label := line[:len(line)-1]
			// Delete label line
			quad = strings.ReplaceAll(quad, line+"\n", "")

			// Replace all label references with the correct line number
			quad = strings.ReplaceAll(quad, label, strconv.Itoa(i-labels+1))
			labels++
		}
	}

	return quad
}
