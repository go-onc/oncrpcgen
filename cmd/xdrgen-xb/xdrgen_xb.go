package main

import (
	"log"
	"os"

	"go.e43.eu/xdr"
	"go.e43.eu/xdrgen/ast"
	"go.e43.eu/xdrgen/internal/genutils"
)

func main() {
	log.SetPrefix("xdrgen-xb")
	log.SetFlags(0)

	f := genutils.ParseFlags(os.Args)

	var spec ast.Specification
	if err := xdr.Read(os.Stdin, &spec); err != nil {
		log.Fatalf("Error reading: %s\n", err)
	}

	of, err := os.Create(f.OutputBasename + ".xb")
	if err != nil {
		log.Fatalf("Error opening output file: %s", err)
	}
	defer of.Close()

	if err := xdr.Write(of, spec); err != nil {
		log.Fatalf("Error writing: %s\n", err)
	}
}
