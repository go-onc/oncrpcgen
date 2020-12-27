package genutils

import (
	"log"
	"strings"

	"github.com/spf13/pflag"
)

type Flags struct {
	InputFilename  string
	OutputBasename string
	Options        []string
}

func (f *Flags) GetOptionValues(name string) []string {
	name = name + "="
	var ours []string
	for _, o := range f.Options {
		if !strings.HasPrefix(o, name) {
			continue
		}

		ours = append(ours, o[len(name):])
	}
	return ours
}

func (f *Flags) ToArgs() []string {
	args := make([]string, 0, 4+2*len(f.Options))
	args = append(args, "-n", f.InputFilename)
	args = append(args, "-o", f.OutputBasename)
	for _, o := range f.Options {
		args = append(args, "-O", o)
	}
	return args
}

func (f *Flags) GetOptionValue(name, def string) string {
	vals := f.GetOptionValues(name)
	if len(vals) == 0 {
		return def
	} else {
		return vals[len(vals)-1]
	}
}

func (f *Flags) Validate() {
	if f.InputFilename == "" {
		log.Fatal("No input filename specified")
	}

	if f.OutputBasename == "" {
		log.Fatal("No output basename specified")
	}
}

func AssociateFlagset(f *Flags, fs *pflag.FlagSet) {
	fs.StringVarP(&f.InputFilename, "input-name", "n", "", "Input filename")
	fs.StringVarP(&f.OutputBasename, "output", "o", "", "Output basename")
	fs.StringSliceVarP(&f.Options, "opt", "O", nil, "User Specified Options")
}

func ParseFlags(args []string) (f Flags) {
	fs := pflag.NewFlagSet(args[0], pflag.ExitOnError)
	AssociateFlagset(&f, fs)
	fs.Parse(args[1:])
	f.Validate()
	return f
}
