package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/nof-sh/CPL-to-QUAD-compiler/cpq"
)

//****************************  Main  ********************************//
func main() {

	fmt.Fprintln(os.Stderr, "CPL to Quad compiler by Nof Shabtay.")
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "USAGE: ./cpq <input-file>")
		return
	}
	if path.Ext(os.Args[1]) != ".ou" {
		fmt.Fprintln(os.Stderr, "Input file extension must be .ou")
		return
	}
	//Read
	infile := os.Args[1]
	code, err := ioutil.ReadFile(infile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot open input CPL file.")
		return
	}
	ast, parseErrors := cpq.Parse(string(code))
	for _, err := range parseErrors {
		fmt.Fprintf(os.Stderr, "ParseError: %s\n", err.Message)
	}
	output, codegenErrors := cpq.Codegen(ast)
	for _, err := range codegenErrors {
		fmt.Fprintf(os.Stderr, "CodegenError: %s\n", err.Message)
	}
	// output QUAD
	if len(parseErrors) == 0 && len(codegenErrors) == 0 {
		// Write file
		outfile := infile[0:len(infile)-3] + ".qud"
		ioutil.WriteFile(outfile, []byte(cpq.RemoveLabels(output)+"\n"+"CPL to Quad compiler by Nof Shabtay."), 0644)
	}
}
