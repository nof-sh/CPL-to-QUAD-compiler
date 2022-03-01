package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

//import (
//	"fmt"
//	"io/ioutil"
//	"os"
//	"path"
//)

var Signature = "CPL to Quad compiler by Nof Shabtay."

func main() {
	fmt.Fprintln(os.Stderr, Signature)

	// Check args
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "USAGE: ./cpq <input-file>")
		return
	}

	// Make sure the input file ends with .ou
	if path.Ext(os.Args[1]) != ".ou" {
		fmt.Fprintln(os.Stderr, "Input file extension must be .ou")
		return
	}

	// Read code file
	infile := os.Args[1]
	code, err := ioutil.ReadFile(infile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot open input CPL file.")
		return
	}

	// Lex & Parse
	ast, parseErrors := cpq.Parse(string(code))
	for _, err := range parseErrors {
		fmt.Fprintf(os.Stderr, "ParseError: %s\n", err.Error())
	}

	// Codegen
	output, codegenErrors := codeGen.Codegen(ast)
	for _, err := range codegenErrors {
		fmt.Fprintf(os.Stderr, "CodegenError: %s\n", err.Error())
	}

	// Generate the filename for the output QUAD file
	if len(parseErrors) == 0 && len(codegenErrors) == 0 {
		// Write output to the QUAD file
		outfile := infile[0:len(infile)-3] + ".qud"
		ioutil.WriteFile(outfile, []byte(codeGen.RemoveLabels(output)+"\n"+Signature), 0644)
	}
}
