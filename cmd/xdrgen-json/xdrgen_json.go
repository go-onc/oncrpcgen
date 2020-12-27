package main

import (
	"encoding/json"
	"log"
	"os"

	"go.e43.eu/xdr"
	"go.e43.eu/xdrgen/ast"
	"go.e43.eu/xdrgen/internal/genutils"
)

func main() {
	log.SetPrefix("xdrgen-json")
	log.SetFlags(0)

	f := genutils.ParseFlags(os.Args)

	var spec ast.Specification
	if err := xdr.Read(os.Stdin, &spec); err != nil {
		log.Fatalf("Error reading: %s\n", err)
	}

	of, err := os.Create(f.OutputBasename + ".json")
	if err != nil {
		log.Fatalf("Error opening output file: %s", err)
	}
	defer of.Close()

	enc := json.NewEncoder(of)
	enc.SetIndent("", "  ")

	if err := enc.Encode(spec); err != nil {
		log.Fatalf("Error writing: %s\n", err)
	}
}
