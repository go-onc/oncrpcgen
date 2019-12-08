package main

import (
	"fmt"
	"io"
	"os"
	"text/scanner"
)

const (
	TokEOF         = scanner.EOF
	TokIdent       = scanner.Ident
	TokIntConst    = scanner.Int
	TokFloatConst  = scanner.Float
	TokCharConst   = scanner.Char
	TokStringConst = scanner.String

	TokBool = -(iota + 16)
	TokCase
	TokConst
	TokDefault
	TokDouble
	TokEnum
	TokFloat
	TokInt
	TokHyper
	TokOpaque
	TokString
	TokStruct
	TokSwitch
	TokTypedef
	TokUnion
	TokUnsigned
	TokVoid
)

var (
	tokToString = map[rune]string{
		TokBool:     "bool",
		TokCase:     "case",
		TokConst:    "const",
		TokDefault:  "default",
		TokDouble:   "double",
		TokEnum:     "enum",
		TokFloat:    "float",
		TokHyper:    "hyper",
		TokInt:      "int",
		TokOpaque:   "opaque",
		TokString:   "string",
		TokStruct:   "struct",
		TokSwitch:   "switch",
		TokTypedef:  "typedef",
		TokUnion:    "union",
		TokUnsigned: "unsigned",
		TokVoid:     "void",
	}
	stringToTok map[string]rune
)

func init() {
	stringToTok = make(map[string]rune)
	for k, v := range tokToString {
		stringToTok[v] = k
	}
}

type Token struct {
	ID       rune
	Value    string
	Position scanner.Position
}

func (t *Token) String() string {
	if s, ok := tokToString[t.ID]; ok {
		return s
	}
	return scanner.TokenString(t.ID)
}

func (t *Token) Error(str string) error {
	return fmt.Errorf("%s: %s\n", t.Position, str)
}

func (t *Token) Errorf(fmts string, params ...interface{}) error {
	return t.Error(fmt.Sprintf(fmts, params...))
}

func (t *Token) Unexpected(ctx string) error {
	return t.Errorf("Unexpected %s while parsing %s", t, ctx)
}

type Lexer struct {
	s  *scanner.Scanner
	nt *Token
}

func NewLexer(rdr io.Reader, filename string) *Lexer {
	l := &Lexer{
		s:  new(scanner.Scanner),
		nt: nil,
	}

	l.s.Init(rdr)
	l.s.Error = func(s *scanner.Scanner, err string) {
		pos := l.s.Position
		if !pos.IsValid() {
			pos = s.Pos()
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", pos, err)
		os.Exit(1)
	}

	l.s.Position.Filename = filename
	return l
}

func (l *Lexer) Position() scanner.Position {
	if l.s.Position.IsValid() {
		return l.s.Position
	} else {
		return l.s.Pos()
	}
}

func (l *Lexer) Peek() *Token {
	if l.nt == nil {
		id := l.s.Scan()

		l.nt = &Token{
			ID:       id,
			Value:    l.s.TokenText(),
			Position: l.s.Position,
		}

		if id == TokIdent {
			if newID, ok := stringToTok[l.nt.Value]; ok {
				l.nt.ID = newID
			}
		}
	}

	return l.nt
}

func (l *Lexer) PeekExpect(ctx string, toks ...rune) (*Token, error) {
	t := l.Peek()
	for _, tok := range toks {
		if tok == t.ID {
			return t, nil
		}
	}

	return nil, t.Unexpected(ctx)
}

func (l *Lexer) Next() *Token {
	t := l.Peek()
	l.nt = nil
	return t
}

func (l *Lexer) NextOneOf(toks ...rune) *Token {
	t := l.Next()
	for _, tok := range toks {
		if tok == t.ID {
			return t
		}
	}

	l.Unget(t)
	return nil
}

func (l *Lexer) Expect(ctx string, toks ...rune) (*Token, error) {
	t := l.Next()
	for _, tok := range toks {
		if tok == t.ID {
			return t, nil
		}
	}

	return nil, t.Unexpected(ctx)
}

func (l *Lexer) Unget(t *Token) {
	if l.nt == nil {
		l.nt = t
	} else {
		panic("attempt to unget a token when there is already one in the buffer")
	}
}
