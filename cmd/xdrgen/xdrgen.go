package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/pflag"
	"go.e43.eu/xdr"
	"go.e43.eu/xdrgen/ast"
	"go.e43.eu/xdrgen/internal/genutils"
	"go.e43.eu/xdrgen/parser"
)

func main() {
	var (
		outDir            string
		enabledGenerators []string
	)
	pflag.StringVarP(&outDir, "output", "O", "", "Output directory (Defaults to same directory as source)")
	pflag.StringSliceVarP(&enabledGenerators, "generators", "G", nil, "Generators to invoke")
	pflag.Parse()

	if len(pflag.Args()) == 0 {
		pflag.Usage()
		return
	}

	if len(enabledGenerators) == 0 {
		log.Fatalf("No generators specified - enable one, e.g. -Gxb, -Ggo, -Gjson")
	}

	var errWg sync.WaitGroup
	errorChan := make(chan error, 5)
	errorCount := 0

	errWg.Add(1)
	go errorWorker(&errWg, &errorCount, errorChan)

	var wg sync.WaitGroup

	wg.Add(1)
	parseCh := make(chan parseResult, 5)
	parseFiles(&wg, parseCh, errorChan, pflag.Args())

	genChans := make([]chan generatorRequest, len(enabledGenerators))
	wg.Add(len(enabledGenerators))
	for i, name := range enabledGenerators {
		i, name := i, name
		genChans[i] = make(chan generatorRequest, 5)
		go generatorWorker(&wg, genChans[i], errorChan, name, nil)
	}

	wg.Add(1)
	go triggerGenerators(&wg, genChans, parseCh, errorChan, outDir)

	wg.Wait()
	close(errorChan)

	errWg.Wait()
	if errorCount > 0 {
		os.Exit(1)
	}
}

type parseResult struct {
	inputName string
	spec      []byte
}

func parseFiles(
	wg *sync.WaitGroup,
	results chan<- parseResult,
	errors chan<- error,
	names []string,
) {
	defer wg.Done()
	defer close(results)

	for _, fname := range names {
		result, err := parseFile(fname)

		if err != nil {
			errors <- err
			return
		} else {
			results <- result
		}
	}
}

func parseFile(fname string) (parseResult, error) {
	inFile, err := os.Open(fname)
	if err != nil {
		return parseResult{}, fmt.Errorf("Error opening '%s': %w", fname, err)
	}
	defer inFile.Close()

	rdr := bufio.NewReader(inFile)
	peeked, _ := rdr.Peek(8)
	isBin := false
	if len(peeked) == 8 {
		isBin = bytes.Equal(peeked, []byte(ast.XDR_BIN_MAGIC_BYTES))
	}

	var spec *ast.Specification
	if !isBin {
		specx, err := parser.ParseSpecification(rdr, fname)
		if err != nil {
			return parseResult{}, fmt.Errorf("Error parsing '%s': %w", fname, err)
		}
		spec = specx
	} else {
		err := xdr.Read(rdr, &spec)
		if err != nil {
			return parseResult{}, fmt.Errorf("Error reading '%s': %w", fname, err)
		}
	}

	specBuf, err := xdr.Marshal(spec)
	if err != nil {
		json, _ := json.Marshal(spec)
		return parseResult{}, fmt.Errorf("Error marshalling result of '%s': %w\n%s", fname, err, string(json))
	}

	return parseResult{
		inputName: fname,
		spec:      specBuf,
	}, nil
}

type generatorRequest struct {
	inputName      string
	outputBasename string
	spec           []byte
}

func triggerGenerators(
	wg *sync.WaitGroup,
	genChans []chan generatorRequest,
	parseResults <-chan parseResult,
	errors chan<- error,
	outDir string,
) {
	defer wg.Done()
	defer func() {
		for _, c := range genChans {
			close(c)
		}
	}()

	for res := range parseResults {
		baseName := strings.TrimRight(res.inputName, filepath.Ext(res.inputName))
		if outDir != "" {
			baseName = filepath.Join(outDir, filepath.Base(baseName))
		}

		for _, c := range genChans {
			c <- generatorRequest{
				inputName:      res.inputName,
				outputBasename: baseName,
				spec:           res.spec,
			}
		}
	}
}

func runGenerator(name string, specSrc io.Reader, flags *genutils.Flags) error {
	cmd := exec.Command("xdrgen-"+name, flags.ToArgs()...)
	cmd.Stdin = specSrc
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Running 'xdrgen-%s': %w", name, err)
	}
	return nil
}

func generatorWorker(
	wg *sync.WaitGroup,
	reqChan <-chan generatorRequest,
	errors chan<- error,
	name string,
	options []string,
) {
	defer wg.Done()

	for req := range reqChan {
		rdr := bytes.NewReader(req.spec)

		flags := &genutils.Flags{
			InputFilename:  req.inputName,
			OutputBasename: req.outputBasename,
			Options:        options,
		}

		if err := runGenerator(name, rdr, flags); err != nil {
			errors <- err
		}
	}
}

func errorWorker(wg *sync.WaitGroup, errorCount *int, errors <-chan error) {
	defer wg.Done()
	for err := range errors {
		log.Print(err)
		*errorCount += 1
	}
}
