package main

import (
	"encoding/json"
	"fmt"
	"os"

	"go.e43.eu/xdr"
	"go.e43.eu/xdrgen/ast"
)

func main() {
	var err error
	in := os.Stdin
	if len(os.Args) > 1 {
		in, err = os.Open(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening '%s': %s\n", os.Args[1], err)
			os.Exit(1)
		}
	}

	var spec ast.Specification
	if err := xdr.Read(in, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading: %s\n", err)
		os.Exit(2)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(spec); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing: %s\n", err)
		os.Exit(3)
	}
}
