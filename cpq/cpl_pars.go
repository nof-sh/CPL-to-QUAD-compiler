package cpq

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type CodeGen struct {
	Errors         []ErrorType
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

//returns new CodeGenerator.
func NewCodeGenerator(output io.Writer) *CodeGen {
	return &CodeGen{
		Errors:         []ErrorType{},
		output:         bufio.NewWriterSize(output, 1),
		Variables:      map[string]DataType{},
		temporaryIndex: 0,
		labelIndex:     0,
		breakStack:     []string{},
	}
}

//generates code to output
func Codegen(program *Program) (string, []ErrorType) {
	buf := new(bytes.Buffer)

	c := NewCodeGenerator(buf)
	c.CodegenProgram(program)

	return buf.String(), c.Errors
}

//generates code for CPL
func (c *CodeGen) CodegenProgram(node *Program) {
	for _, declaration := range node.Declarations {
		for _, name := range declaration.Names {
			if _, exists := c.Variables[name]; exists {
				c.Errors = append(c.Errors, ErrorType{
					Message: fmt.Sprintf("variable %s already defined", name),
					Pos:     declaration.Pos,
				})
				continue
			}
			c.Variables[name] = declaration.Type
		}
	}
	c.CodegenStatement(node.StatementsBlock)
	c.output.WriteString("HALT\n")
}

//generates code for CPL
func (c *CodeGen) CodegenStatement(node Statement) {
	switch s := node.(type) {
	case *Assignment:
		c.CodegenAssignmentStatement(s)
	case *Input:
		c.CodegenInputStatement(s)
	case *Output:
		c.CodegenOutputStatement(s)
	case *IfStatement:
		c.CodegenIfStatement(s)
	case *WhileStatement:
		c.CodegenWhileStatement(s)
	case *Switch:
		c.CodegenSwitchStatement(s)
	case *Break:
		c.CodegenBreakStatement(s)
	case *Block:
		c.CodegenStatementsBlock(s)
	}
}

//generates code for assignment
func (c *CodeGen) CodegenAssignmentStatement(node *Assignment) {
	exp := c.CodegenExpression(node)
	if _, exists := c.Variables[node.Variable]; !exists {
		c.Errors = append(c.Errors, ErrorType{
			Message: fmt.Sprintf("undefined variable %s", node.Variable),
			Pos:     node.Pos,
		})
		return
	}
	if exp == nil {
		return
	}
	if node.CastType != Unknown && node.CastType != exp.Type {
		exp = c.codegenCastExpression(exp, node.CastType)
	}
	if c.Variables[node.Variable] == Integer && exp.Type == Float {
		c.Errors = append(c.Errors, ErrorType{
			Message: fmt.Sprintf("cannot assign float value to int variable %s", node.Variable),
			Pos:     node.Pos,
		})
		return
	}
	if c.Variables[node.Variable] == Float && exp.Type == Integer {
		exp = c.codegenCastExpression(exp, Float)
	}
	if c.Variables[node.Variable] == Integer {
		c.output.WriteString(fmt.Sprintf("IASN %s %s\n", node.Variable, exp.Code))
	} else if c.Variables[node.Variable] == Float {
		c.output.WriteString(fmt.Sprintf("RASN %s %s\n", node.Variable, exp.Code))
	}
}

//generates code for input
func (c *CodeGen) CodegenInputStatement(node *Input) {
	if _, exists := c.Variables[node.Variable]; !exists {
		c.Errors = append(c.Errors, ErrorType{
			Message: fmt.Sprintf("undefined variable %s", node.Variable),
			Pos:     node.Pos,
		})
		return
	}
	if c.Variables[node.Variable] == Integer {
		c.output.WriteString(fmt.Sprintf("IINP %s\n", node.Variable))
	} else if c.Variables[node.Variable] == Float {
		c.output.WriteString(fmt.Sprintf("RINP %s\n", node.Variable))
	}
}

//generates code for output
func (c *CodeGen) CodegenOutputStatement(node *Output) {
	exp := c.CodegenExpression(node)
	if exp == nil {
		return
	}
	if exp.Type == Integer {
		c.output.WriteString(fmt.Sprintf("IPRT %s\n", exp.Code))
	} else if exp.Type == Float {
		c.output.WriteString(fmt.Sprintf("RPRT %s\n", exp.Code))
	}
}

//generates code for 'if'
func (c *CodeGen) CodegenIfStatement(node *IfStatement) {
	condition := c.CodegenBooleanExpression(node.Condition)
	endIfLabel := c.getNewLabel()
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

//generates code for while
func (c *CodeGen) CodegenWhileStatement(node *WhileStatement) {
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

//generates code for switch
func (c *CodeGen) CodegenSwitchStatement(node *Switch) {
	exp := c.CodegenExpression(node)
	if exp == nil {
		return
	}
	if exp.Type != Integer {
		c.Errors = append(c.Errors, ErrorType{
			Message: "switch expression must be an integer",
			Pos:     node.Position,
		})
	}
	temp := c.getTemp()
	caseLabels := map[int]string{}
	for i, switchCase := range node.Cases {
		caseLabels[i] = c.getNewLabel()
		c.output.WriteString(fmt.Sprintf("INQL %s %s %d\n", temp, exp.Code, switchCase.Value))
		c.output.WriteString(fmt.Sprintf("JMPZ %s %s\n", caseLabels[i], temp))
	}
	defaultLabel := c.getNewLabel()
	endSwitchLabel := c.getNewLabel()
	c.output.WriteString(fmt.Sprintf("JUMP %s\n", defaultLabel))
	c.breakStack = append(c.breakStack, endSwitchLabel)
	for i, switchCase := range node.Cases {
		c.output.WriteString(fmt.Sprintf("%s:\n", caseLabels[i]))
		c.CodegenStatement(&Block{
			Statements: switchCase.Statements,
		})
	}
	c.output.WriteString(fmt.Sprintf("%s:\n", defaultLabel))
	c.CodegenStatement(&Block{
		Statements: node.DefaultCase,
	})
	if c.breakStack[len(c.breakStack)-1] == endSwitchLabel {
		c.breakStack = c.breakStack[:len(c.breakStack)-1]
	}
	c.output.WriteString(fmt.Sprintf("%s:\n", endSwitchLabel))
}

// generates code for break
func (c *CodeGen) CodegenBreakStatement(node *Break) {
	if len(c.breakStack) == 0 {
		c.Errors = append(c.Errors, ErrorType{
			Message: "break statement must be inside a while loop or a switch case",
			Pos:     node.Position,
		})
		return
	}
	c.output.WriteString(fmt.Sprintf("JUMP %s\n", c.breakStack[len(c.breakStack)-1]))
}

//generates code for block.
func (c *CodeGen) CodegenStatementsBlock(node *Block) {
	for _, statement := range node.Statements {
		c.CodegenStatement(statement)
	}
}

// generates code for CPL
func (c *CodeGen) CodegenExpression(node Node) *Expression {
	switch temp := node.(type) {
	case *Arithmetic:
		return c.CodegenArithmeticExpression(temp)
	case *Variable:
		return c.CodegenVariableExpression(temp)
	case *FloatNum:
		return c.CodegenFloatLiteral(temp)
	case *IntNum:
		return c.CodegenIntLiteral(temp)
	}
	return nil
}

//generates code for an arithmetic
func (c *CodeGen) CodegenArithmeticExpression(aryth *Arithmetic) *Expression {
	lhs := c.CodegenExpression(aryth)
	rhs := c.CodegenExpression(aryth)
	if lhs == nil || rhs == nil {
		return nil
	}
	result := &Expression{
		Code: c.getTemp(),
		Type: calculateExpressionType(lhs.Type, rhs.Type),
	}
	if result.Type == Float {
		lhs = c.codegenCastExpression(lhs, Float)
		rhs = c.codegenCastExpression(rhs, Float)
	}
	switch aryth.Operator {
	case Add:
		if result.Type == Integer {
			c.output.WriteString(fmt.Sprintf("IADD %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		} else if result.Type == Float {
			c.output.WriteString(fmt.Sprintf("RADD %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		}
	case Subtract:
		if result.Type == Integer {
			c.output.WriteString(fmt.Sprintf("ISUB %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		} else if result.Type == Float {
			c.output.WriteString(fmt.Sprintf("RSUB %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		}
	case Multiply:
		if result.Type == Integer {
			c.output.WriteString(fmt.Sprintf("IMLT %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		} else if result.Type == Float {
			c.output.WriteString(fmt.Sprintf("RMLT %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		}
	case Divide:
		if result.Type == Integer {
			c.output.WriteString(fmt.Sprintf("IDIV %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		} else if result.Type == Float {
			c.output.WriteString(fmt.Sprintf("RDIV %s %s %s\n", result.Code, lhs.Code, rhs.Code))
		}
	}
	return result
}

//generates code for variable
func (c *CodeGen) CodegenVariableExpression(node *Variable) *Expression {
	if _, exists := c.Variables[node.Variable]; !exists {
		c.Errors = append(c.Errors, ErrorType{
			Message: fmt.Sprintf("undefined variable %s", node.Variable),
			Pos:     node.Position,
		})
		return nil
	}
	return &Expression{Code: node.Variable, Type: c.Variables[node.Variable]}
}

//generates code for integer
func (c *CodeGen) CodegenIntLiteral(node *IntNum) *Expression {
	return &Expression{
		Code: fmt.Sprintf("%d", node.Value),
		Type: Integer,
	}
}

//generates code for float
func (c *CodeGen) CodegenFloatLiteral(node *FloatNum) *Expression {
	return &Expression{
		Code: fmt.Sprintf("%f", node.Value),
		Type: Float,
	}
}

func (c *CodeGen) CodegenBooleanExpression(node Boolean) string {
	switch s := node.(type) {
	case *Or:
		return c.CodegenOrBooleanExpression(s)
	case *And:
		return c.CodegenAndBooleanExpression(s)
	case *Not:
		return c.CodegenNotBooleanExpression(s)
	case *Compare:
		return c.CodegenCompareBooleanExpression(s)
	}
	return ""
}

//generates code for OR
func (c *CodeGen) CodegenOrBooleanExpression(node *Or) string {
	lhs := c.CodegenBooleanExpression(node.LHS)
	rhs := c.CodegenBooleanExpression(node.RHS)
	if lhs == "" || rhs == "" {
		return ""
	}
	result := c.getTemp()
	c.output.WriteString(fmt.Sprintf("IADD %s %s %s\n", result, lhs, rhs))
	c.output.WriteString(fmt.Sprintf("IGRT %s %s 0\n", result, result))
	return result
}

//generates code for AND
func (c *CodeGen) CodegenAndBooleanExpression(node *And) string {
	lhs := c.CodegenBooleanExpression(node.LHS)
	rhs := c.CodegenBooleanExpression(node.RHS)
	if lhs == "" || rhs == "" {
		return ""
	}
	result := c.getTemp()
	c.output.WriteString(fmt.Sprintf("IMLT %s %s %s\n", result, lhs, rhs))
	return result
}

//generates code for NOT
func (c *CodeGen) CodegenNotBooleanExpression(node *Not) string {
	value := c.CodegenBooleanExpression(node.Value)
	if value == "" {
		return ""
	}
	result := c.getTemp()
	c.output.WriteString(fmt.Sprintf("ISUB %s 1 %s\n", result, value))
	return result
}

//generates code for comparison
func (c *CodeGen) CodegenCompareBooleanExpression(node *Compare) string {
	if node.Operator == GreaterThanOrEqualTo {
		return c.CodegenOrBooleanExpression(&Or{
			LHS: &Compare{
				LHS:      node.LHS,
				Operator: EqualTo,
				RHS:      node.RHS,
			},
			RHS: &Compare{
				LHS:      node.LHS,
				Operator: GreaterThan,
				RHS:      node.RHS,
			},
		})
	}
	if node.Operator == LessThenOrEqualTo {
		return c.CodegenOrBooleanExpression(&Or{
			LHS: &Compare{
				LHS:      node.LHS,
				Operator: EqualTo,
				RHS:      node.RHS,
			},
			RHS: &Compare{
				LHS:      node.LHS,
				Operator: LessThan,
				RHS:      node.RHS,
			},
		})
	}
	lhs := c.CodegenExpression(node)
	rhs := c.CodegenExpression(node)
	if lhs == nil || rhs == nil {
		return ""
	}
	compareType := calculateExpressionType(lhs.Type, rhs.Type)

	if compareType == Float {
		lhs = c.codegenCastExpression(lhs, Float)
		rhs = c.codegenCastExpression(rhs, Float)
	}
	result := c.getTemp()
	switch node.Operator {
	case EqualTo:
		if compareType == Integer {
			c.output.WriteString(fmt.Sprintf("IEQL %s %s %s\n", result, lhs.Code, rhs.Code))
		} else if compareType == Float {
			c.output.WriteString(fmt.Sprintf("REQL %s %s %s\n", result, lhs.Code, rhs.Code))
		}
	case NotEqualTo:
		if compareType == Integer {
			c.output.WriteString(fmt.Sprintf("INQL %s %s %s\n", result, lhs.Code, rhs.Code))
		} else if compareType == Float {
			c.output.WriteString(fmt.Sprintf("RNQL %s %s %s\n", result, lhs.Code, rhs.Code))
		}
	case GreaterThan:
		if compareType == Integer {
			c.output.WriteString(fmt.Sprintf("IGRT %s %s %s\n", result, lhs.Code, rhs.Code))
		} else if compareType == Float {
			c.output.WriteString(fmt.Sprintf("RGRT %s %s %s\n", result, lhs.Code, rhs.Code))
		}
	case LessThan:
		if compareType == Integer {
			c.output.WriteString(fmt.Sprintf("ILSS %s %s %s\n", result, lhs.Code, rhs.Code))
		} else if compareType == Float {
			c.output.WriteString(fmt.Sprintf("RLSS %s %s %s\n", result, lhs.Code, rhs.Code))
		}
	}
	return result
}

func (c *CodeGen) getTemp() string {
	c.temporaryIndex++
	return fmt.Sprintf("_t%d", c.temporaryIndex)
}

func (c *CodeGen) getNewLabel() string {
	c.labelIndex++
	return fmt.Sprintf("@%d", c.labelIndex)
}

func (c *CodeGen) codegenCastExpression(exp *Expression, targetType DataType) *Expression {
	if exp.Type == targetType {
		return exp
	}
	result := &Expression{
		Code: c.getTemp(),
		Type: targetType,
	}
	switch targetType {
	case Integer:
		c.output.WriteString(fmt.Sprintf("RTOI %s %s\n", result.Code, exp.Code))
	case Float:
		c.output.WriteString(fmt.Sprintf("ITOR %s %s\n", result.Code, exp.Code))
	default:
		panic("Invalid type!")
	}
	return result
}

func calculateExpressionType(types ...DataType) DataType {
	for _, t := range types {
		if t == Float {
			return Float
		}
	}

	return Integer
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
