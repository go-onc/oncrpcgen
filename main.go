package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	xdr3 "github.com/stellar/go-xdr/xdr3"
	"go.e43.eu/go-onc/oncrpcgen/ast"
)

type Generator func(baseName string, spec *ast.Specification) error

func xbGenerator(baseName string, spec *ast.Specification) error {
	fileName := baseName + ".xb"
	log.Println("Generating ", fileName)

	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("Error opening %s: %v", fileName, err)
	}

	_, err = xdr3.Marshal(file, spec)
	return err
}

func jsonGenerator(baseName string, spec *ast.Specification) error {
	json, err := json.MarshalIndent(spec, "", "    ")
	if err != nil {
		return fmt.Errorf("Marshalling JSON: %c", err)
	}

	fileName := baseName + ".json"
	log.Println("Generating ", fileName)
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("Error opening %s: %v", fileName, err)
	}

	_, err = file.Write(json)
	return err
}

func goGenerator(baseName string, spec *ast.Specification) error {
	fileName := baseName + ".x.go"
	log.Println("Generating ", fileName)

	buf, err := GenSpecification(spec)
	if err != nil {
		return err
	}

	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("Error opening %s: %v", fileName, err)
	}

	_, err = file.Write(buf)
	return err
}

var Generators = map[string]Generator{
	"xb":   xbGenerator,
	"json": jsonGenerator,
	"go":   goGenerator,
}

func main() {
	var (
		outDir            string
		enabledGenerators []string
	)
	pflag.StringVarP(&outDir, "output", "O", "", "Output directory (Defaults to same directory as source)")
	pflag.StringSliceVarP(&enabledGenerators, "generators", "G", nil, "Generators to apply")
	pflag.Parse()

	if len(pflag.Args()) == 0 {
		pflag.Usage()
		return
	}

	if len(enabledGenerators) == 0 {
		log.Fatalf("No generators specified - enable one, e.g. -Gxb, -Ggo, -Gjson")
	}

	var generators []Generator
	for _, gName := range enabledGenerators {
		g, ok := Generators[gName]
		if !ok {
			log.Fatalf("No such generator '%s'", gName)
		}
		generators = append(generators, g)
	}

	for _, inFileName := range pflag.Args() {
		baseName := strings.TrimRight(inFileName, filepath.Ext(inFileName))
		if outDir != "" {
			baseName = filepath.Join(outDir, filepath.Base(baseName))
		}

		inFile, err := os.Open(inFileName)
		if err != nil {
			log.Fatalf("Error opening %s: %v", inFileName, err)
		}

		lex := NewLexer(inFile, inFile.Name())
		spec, err := ParseSpecification(lex)
		if err != nil {
			log.Fatalf("Error parsing %s: %v", inFileName, err)
		}

		for i, g := range generators {
			if err := g(baseName, spec); err != nil {
				log.Fatalf("Error running %s generator on %s: %v", enabledGenerators[i], inFileName, err)
			}
		}
	}
}
