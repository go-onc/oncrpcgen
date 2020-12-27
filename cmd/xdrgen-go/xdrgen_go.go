package main

import (
	"log"
	"os"

	"go.e43.eu/xdr"
	"go.e43.eu/xdrgen/ast"
	"go.e43.eu/xdrgen/internal/gengo"
	"go.e43.eu/xdrgen/internal/genutils"
)

func main() {
	log.SetPrefix("xdrgen-go")
	log.SetFlags(0)

	f := genutils.ParseFlags(os.Args)

	var spec ast.Specification
	if err := xdr.Read(os.Stdin, &spec); err != nil {
		log.Fatalf("Error reading: %s\n", err)
	}

	buf, err := gengo.GenSpecification(&spec)
	if err != nil {
		log.Fatalf("Error generating: %s\n", err)
	}

	of, err := os.Create(f.OutputBasename + ".x.go")
	if err != nil {
		log.Fatalf("Error opening output file: %s", err)
	}
	defer of.Close()

	if _, err = of.Write(buf); err != nil {
		log.Fatalf("Error writing output file: %s", err)
	}
}
