package main

import (
	"strconv"

	"go.e43.eu/go-onc/oncrpcgen/ast"
)

func ParseSpecification(l *Lexer) (*ast.Specification, error) {
	s := new(ast.Specification)
	s.Magic = ast.XDR_BIN_MAGIC

	if l.Peek().ID == '#' {
		l.Next()
		a, err := ParseAttributes(s, l)
		if err != nil {
			return nil, err
		}
		s.Attributes = a
	}

	for t := l.Peek(); t.ID != TokEOF; t = l.Peek() {
		d, err := ParseDefinition(s, l)
		if err != nil {
			return nil, err
		}

		if err := s.PutDefinition(d); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func ParseAttributes(s *ast.Specification, l *Lexer) (map[string]*ast.Constant, error) {
	if l.NextOneOf('[') == nil {
		return nil, nil
	}

	a := make(map[string]*ast.Constant)
	for {
		if t := l.NextOneOf(',', ']'); t != nil {
			if t.ID == ',' {
				continue
			} else if t.ID == ']' {
				return a, nil
			}
		}

		ident, err := l.Expect("attributes", TokIdent)
		if err != nil {
			return nil, err
		}

		t := l.Next()
		switch t.ID {
		case ']':
			return a, nil
		case ',':
			a[ident.Value] = &ast.Constant{
				Type:  ast.CONST_BOOL,
				VBool: true,
			}
			continue
		case '(':
			val, err := ParseValue(s, l)
			if err != nil {
				return nil, err
			}

			a[ident.Value] = val

			if _, err := l.Expect("attribute", ')'); err != nil {
				return nil, err
			}

			t = l.Next()
			switch t.ID {
			case ',':
				continue
			case ']':
				return a, nil
			default:
				return nil, t.Unexpected("attributes")
			}
		default:
			return nil, t.Unexpected("attributes")
		}
	}
}

func ParseDefinition(s *ast.Specification, l *Lexer) (d *ast.Definition, err error) {
	a, err := ParseAttributes(s, l)
	if err != nil {
		return nil, err
	}

	t := l.Peek()
	switch t.ID {
	case TokTypedef:
		d, err = ParseTypedef(s, l)
	case TokEnum:
		d, err = ParseEnum(s, l)
	case TokStruct:
		d, err = ParseStruct(s, l)
	case TokUnion:
		d, err = ParseUnion(s, l)
	case TokConst:
		d, err = ParseConst(s, l)
	default:
		err = t.Unexpected("definition")
	}

	if err != nil {
		return nil, err
	}
	d.Attributes = a
	return d, nil
}

func ParseValue(s *ast.Specification, l *Lexer) (*ast.Constant, error) {
	t := l.Next()
	switch t.ID {
	case TokIdent:
		return s.GetConstant(t.Value)

	case TokIntConst:
		istr := t.Value
		negative := false
		if istr[0] == '-' {
			negative = true
			istr = istr[1:]
		}

		ui, err := strconv.ParseUint(istr, 0, 64)
		if err != nil {
			return nil, t.Error(err.Error())
		}

		if negative {
			return &ast.Constant{
				Type:    ast.CONST_NEG_INT,
				VPosInt: ui,
			}, nil
		} else {
			return &ast.Constant{
				Type:    ast.CONST_POS_INT,
				VPosInt: ui,
			}, nil
		}

	case TokFloatConst:
		f, err := strconv.ParseFloat(t.Value, 64)
		if err != nil {
			return nil, t.Error(err.Error())
		}

		return &ast.Constant{
			Type:   ast.CONST_FLOAT,
			VFloat: f,
		}, nil

	case TokStringConst:
		str, err := strconv.Unquote(t.Value)
		if err != nil {
			return nil, t.Error(err.Error())
		}
		return &ast.Constant{
			Type:    ast.CONST_STRING,
			VString: str,
		}, nil

	default:
		return nil, t.Unexpected("constant")
	}
}

func ParseConst(s *ast.Specification, l *Lexer) (*ast.Definition, error) {
	if _, err := l.Expect("const", TokConst); err != nil {
		return nil, err
	}

	ident, err := l.Expect("const", TokIdent)
	if err != nil {
		return nil, err
	}

	if _, err := l.Expect("const", '='); err != nil {
		return nil, err
	}

	val, err := ParseValue(s, l)
	if err != nil {
		return nil, err
	}

	if _, err := l.Expect("const", ';'); err != nil {
		return nil, err
	}

	return &ast.Definition{
		Name: ident.Value,
		Body: &ast.Definition_Body{
			Kind:     ast.DEFINITION_KIND_CONSTANT,
			Constant: val,
		},
	}, nil
}

func ParseTypedef(s *ast.Specification, l *Lexer) (*ast.Definition, error) {
	if _, err := l.Expect("typedef", TokTypedef); err != nil {
		return nil, err
	}

	decl, err := ParseDeclaration(s, l)
	if err != nil {
		return nil, err
	}

	if _, err := l.Expect("typedef", ';'); err != nil {
		return nil, err
	}

	return &ast.Definition{
		Name: decl.Name,
		Body: &ast.Definition_Body{
			Kind: ast.DEFINITION_KIND_TYPE,
			Type: &ast.Type{
				Kind:    ast.TYPE_TYPEDEF,
				TypeDef: decl,
			},
		},
	}, nil
}

func ParseDeclaration(s *ast.Specification, l *Lexer) (*ast.Declaration, error) {
	var err error
	d := &ast.Declaration{
		Modifier: new(ast.Declaration_Modifier),
	}

	d.Attributes, err = ParseAttributes(s, l)
	if err != nil {
		return nil, err
	}

	t := l.Next()
	switch t.ID {
	case TokUnsigned:
		t, err := l.Expect("declaration", TokInt, TokHyper)
		if err != nil {
			return nil, err
		}

		switch t.ID {
		case TokInt:
			d.Type = ast.UnsignedInt()
		case TokHyper:
			d.Type = ast.UnsignedHyper()
		}
	case TokInt:
		d.Type = ast.Int()
	case TokHyper:
		d.Type = ast.Hyper()
	case TokFloat:
		d.Type = ast.Float()
	case TokDouble:
		d.Type = ast.Double()
	case TokBool:
		d.Type = ast.Bool()
	case TokString:
		d.Type = ast.String()
	case TokOpaque:
		d.Type = ast.Opaque()
	case TokEnum:
		l.Unget(t)
		d.Type, err = ParseEnumTypeSpec(s, l)
		if err != nil {
			return nil, err
		}
	case TokStruct:
		l.Unget(t)
		d.Type, err = ParseStructTypeSpec(s, l)
		if err != nil {
			return nil, err
		}
	case TokUnion:
		l.Unget(t)
		d.Type, err = ParseUnionTypeSpec(s, l)
		if err != nil {
			return nil, err
		}
	case TokIdent:
		d.Type = ast.Ref(t.Value)
	case TokVoid:
		d.Type = ast.Void()
		return d, nil
	default:
		return nil, t.Unexpected("declaration")
	}

	if l.NextOneOf('*') != nil {
		d.Modifier.Kind = ast.DECLARATION_MODIFIER_OPTIONAL
	}

	t, err = l.Expect("declaration", TokIdent)
	if err != nil {
		return nil, err
	}
	d.Name = t.Value

	var t2 *Token
	if d.Modifier.Kind != ast.DECLARATION_MODIFIER_OPTIONAL {
		t2 = l.NextOneOf('<', '[')
	}

	switch {
	case t2 != nil && t2.ID == '<':
		if l.Peek().ID == '>' {
			d.Modifier.Kind = ast.DECLARATION_MODIFIER_UNBOUNDED
		} else {
			val, err := ParseValue(s, l)
			if err != nil {
				return nil, err
			}

			vu32, err := val.AsU32()
			if err != nil {
				return nil, err
			}

			d.Modifier.Kind = ast.DECLARATION_MODIFIER_FLEXIBLE
			d.Modifier.Size = vu32
		}
		if _, err := l.Expect("declaration", '>'); err != nil {
			return nil, err
		}
	case t2 != nil && t2.ID == '[' && d.Type.Kind != ast.TYPE_STRING:
		val, err := ParseValue(s, l)
		if err != nil {
			return nil, err
		}

		vu32, err := val.AsU32()
		if err != nil {
			return nil, err
		}

		d.Modifier.Kind = ast.DECLARATION_MODIFIER_FIXED
		d.Modifier.Size = vu32
		if _, err := l.Expect("declaration", ']'); err != nil {
			return nil, err
		}
	case d.Type.Kind == ast.TYPE_STRING || d.Type.Kind == ast.TYPE_OPAQUE:
		return nil, t.Errorf("string or opaque must have size specifier")
	}

	return d, nil
}

func ParseEnumTypeSpec(s *ast.Specification, l *Lexer) (*ast.Type, error) {
	l.Expect("enum", TokEnum)
	return ParseEnumBody(s, l)
}

func ParseEnum(s *ast.Specification, l *Lexer) (*ast.Definition, error) {
	if _, err := l.Expect("enum", TokEnum); err != nil {
		return nil, err
	}

	ident, err := l.Expect("enum", TokIdent)
	if err != nil {
		return nil, err
	}

	body, err := ParseEnumBody(s, l)
	if err != nil {
		return nil, err
	}

	if _, err := l.Expect("enum", ';'); err != nil {
		return nil, err
	}

	return &ast.Definition{
		Name: ident.Value,
		Body: &ast.Definition_Body{
			Kind: ast.DEFINITION_KIND_TYPE,
			Type: body,
		},
	}, nil
}

func ParseEnumBody(s *ast.Specification, l *Lexer) (*ast.Type, error) {
	es := new(ast.EnumSpec)

	if _, err := l.Expect("enum", '{'); err != nil {
		return nil, err
	}

body:
	for {
		t, err := l.Expect("enum body", TokIdent, ',', '}')
		if err != nil {
			return nil, err
		}

		switch t.ID {
		case TokIdent:
			if es.HasOption(t.Value) {
				t.Errorf("Name '%s' already defined", t.Value)
			}

			if _, err := l.Expect("enum body", '='); err != nil {
				return nil, err
			}
			v, err := ParseValue(s, l)
			if err != nil {
				return nil, err
			}
			vu32, err := v.AsU32()
			if err != nil {
				return nil, err
			}

			es.Options = append(es.Options, &ast.EnumSpec_Options{
				Name:  t.Value,
				Value: vu32,
			})

			s.PutDefinition(&ast.Definition{
				Name: t.Value,
				Body: &ast.Definition_Body{
					Kind: ast.DEFINITION_KIND_CONSTANT,
					Constant: &ast.Constant{
						Type:  ast.CONST_ENUM,
						VEnum: vu32,
					},
				},
			})

			t, err = l.Expect("enum body", ',', '}')
			if err != nil {
				return nil, err
			}
		}

		switch t.ID {
		case ',':
			continue body

		case '}':
			break body

		default:
			t.Unexpected("enum body")
		}
	}

	return &ast.Type{
		Kind:     ast.TYPE_ENUM,
		EnumSpec: es,
	}, nil
}

func ParseStruct(s *ast.Specification, l *Lexer) (*ast.Definition, error) {
	if _, err := l.Expect("struct", TokStruct); err != nil {
		return nil, err
	}

	ident, err := l.Expect("struct", TokIdent)
	if err != nil {
		return nil, err
	}

	body, err := ParseStructBody(s, l)
	if err != nil {
		return nil, err
	}

	if _, err := l.Expect("struct", ';'); err != nil {
		return nil, err
	}

	return &ast.Definition{
		Name: ident.Value,
		Body: &ast.Definition_Body{
			Kind: ast.DEFINITION_KIND_TYPE,
			Type: body,
		},
	}, nil
}

func ParseStructTypeSpec(s *ast.Specification, l *Lexer) (*ast.Type, error) {
	if _, err := l.Expect("struct", TokStruct); err != nil {
		return nil, err
	}

	return ParseStructBody(s, l)
}

func ParseStructBody(s *ast.Specification, l *Lexer) (*ast.Type, error) {
	if _, err := l.Expect("struct", '{'); err != nil {
		return nil, err
	}

	ss := new(ast.StructSpec)
body:
	for {
		if l.NextOneOf('}') != nil {
			break body
		}

		t := l.Peek()
		decl, err := ParseDeclaration(s, l)
		if err != nil {
			return nil, err
		}

		if ss.HasMember(decl.Name) {
			return nil, t.Errorf("Attempt to redefine '%s'", decl.Name)
		}

		if _, err := l.Expect("struct body", ';'); err != nil {
			return nil, err
		}

		ss.Members = append(ss.Members, decl)
	}

	return &ast.Type{
		Kind:       ast.TYPE_STRUCT,
		StructSpec: ss,
	}, nil
}

func ParseUnion(s *ast.Specification, l *Lexer) (*ast.Definition, error) {
	if _, err := l.Expect("union", TokUnion); err != nil {
		return nil, err
	}

	ident, err := l.Expect("union", TokIdent)
	if err != nil {
		return nil, err
	}

	body, err := ParseUnionBody(s, l)
	if err != nil {
		return nil, err
	}
	if _, err := l.Expect("union", ';'); err != nil {
		return nil, err
	}

	return &ast.Definition{
		Name: ident.Value,
		Body: &ast.Definition_Body{
			Kind: ast.DEFINITION_KIND_TYPE,
			Type: body,
		},
	}, nil
}

func ParseUnionTypeSpec(s *ast.Specification, l *Lexer) (*ast.Type, error) {
	l.Expect("union", TokUnion)
	return ParseUnionBody(s, l)
}

func ParseUnionBody(s *ast.Specification, l *Lexer) (*ast.Type, error) {
	var err error
	us := &ast.UnionSpec{
		Options: make(map[uint32]uint32),
	}

	if _, err := l.Expect("union", TokSwitch); err != nil {
		return nil, err
	}
	if _, err := l.Expect("union", '('); err != nil {
		return nil, err
	}

	us.Discriminant, err = ParseDeclaration(s, l)
	if err != nil {
		return nil, err
	}

	if _, err := l.Expect("union", ')'); err != nil {
		return nil, err
	}
	if _, err := l.Expect("union", '{'); err != nil {
		return nil, err
	}

body:
	for {
		switch l.Peek().ID {
		case TokDefault, '}':
			break body
		}

		caseTok, err := l.Expect("union", TokCase)
		if err != nil {
			return nil, err
		}
		discriminantVal, err := ParseValue(s, l)
		if err != nil {
			return nil, err
		}
		discriminant, err := discriminantVal.AsU32()
		if err != nil {
			return nil, err
		}
		if _, err := l.Expect("union", ':'); err != nil {
			return nil, err
		}
		declaration, err := ParseDeclaration(s, l)
		if err != nil {
			return nil, err
		}
		if _, err := l.Expect("union", ';'); err != nil {
			return nil, err
		}

		if us.HasOption(discriminant) {
			return nil, caseTok.Errorf("Value conflicts with existing alternative")
		} else if declaration.Name == us.Discriminant.Name {
			return nil, caseTok.Errorf("Alternative name '%s' conflicts with that of union discriminant", declaration.Name)
		}

		pos, existing := us.GetMember(declaration.Name)
		if existing != nil {
			if !existing.Equal(declaration) {
				caseTok.Error("Alternative name is not unique and type mismatches")
			}
		} else {
			pos = uint32(len(us.Members))
			us.Members = append(us.Members, declaration)
		}

		us.Options[discriminant] = pos
	}

	if caseTok := l.NextOneOf(TokDefault); caseTok != nil {
		if _, err := l.Expect("union", ':'); err != nil {
			return nil, err
		}
		declaration, err := ParseDeclaration(s, l)
		if err != nil {
			return nil, err
		}
		if _, err := l.Expect("union", ';'); err != nil {
			return nil, err
		}

		if declaration.Name == us.Discriminant.Name {
			return nil, caseTok.Errorf("Alternative name conflicts with that of union discriminant")
		}

		pos, existing := us.GetMember(declaration.Name)
		if existing != nil {
			if !existing.Equal(declaration) {
				caseTok.Error("Alternative name is not unique and type mismatches")
			}
		} else {
			pos = uint32(len(us.Members))
			us.Members = append(us.Members, declaration)
		}

		us.DefaultMember = &pos
	}

	if _, err := l.Expect("union", '}'); err != nil {
		return nil, err
	}

	return &ast.Type{
		Kind:      ast.TYPE_UNION,
		UnionSpec: us,
	}, nil
}
