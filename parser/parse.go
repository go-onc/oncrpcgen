package parser

import (
	"io"
	"strconv"

	"go.e43.eu/xdrgen/ast"
	"go.e43.eu/xdrgen/internal/lexer"
)

func ParseSpecification(rdr io.Reader, filename string) (*ast.Specification, error) {
	l := lexer.NewLexer(rdr, filename)
	return parseSpecification(l)
}

func parseSpecification(l *lexer.Lexer) (*ast.Specification, error) {
	s := new(ast.Specification)
	s.Magic = ast.XDR_BIN_MAGIC

	if l.Peek().ID == '#' {
		l.Next()
		a, err := parseAttributes(s, l)
		if err != nil {
			return nil, err
		}
		s.Attributes = a
	}

	for t := l.Peek(); t.ID != lexer.TokEOF; t = l.Peek() {
		d, err := parseDefinition(s, l)
		if err != nil {
			return nil, err
		}

		if _, err := s.PutDefinition(d); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func parseAttributes(s *ast.Specification, l *lexer.Lexer) (ast.Attributes, error) {
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

		ident, err := l.Expect("attributes", lexer.TokIdent)
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
			val, err := parseValue(s, l)
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
				return ast.Attributes(a), nil
			default:
				return nil, t.Unexpected("attributes")
			}
		default:
			return nil, t.Unexpected("attributes")
		}
	}
}

func parseDefinition(s *ast.Specification, l *lexer.Lexer) (d *ast.Definition, err error) {
	a, err := parseAttributes(s, l)
	if err != nil {
		return nil, err
	}

	t := l.Peek()
	switch t.ID {
	case lexer.TokTypedef:
		d, err = parseTypedef(s, l)
	case lexer.TokEnum:
		d, err = parseEnum(s, l)
	case lexer.TokStruct:
		d, err = parseStruct(s, l)
	case lexer.TokUnion:
		d, err = ParseUnion(s, l)
	case lexer.TokConst:
		d, err = parseConst(s, l)
	default:
		err = t.Unexpected("definition")
	}

	if err != nil {
		return nil, err
	}
	d.Attributes = a
	return d, nil
}

func parseValue(s *ast.Specification, l *lexer.Lexer) (*ast.Constant, error) {
	t := l.Next()
	switch t.ID {
	case lexer.TokIdent:
		return s.GetConstant(t.Value)

	case lexer.TokIntConst:
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

	case lexer.TokFloatConst:
		f, err := strconv.ParseFloat(t.Value, 64)
		if err != nil {
			return nil, t.Error(err.Error())
		}

		return &ast.Constant{
			Type:   ast.CONST_FLOAT,
			VFloat: f,
		}, nil

	case lexer.TokStringConst:
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

func parseConst(s *ast.Specification, l *lexer.Lexer) (*ast.Definition, error) {
	if _, err := l.Expect("const", lexer.TokConst); err != nil {
		return nil, err
	}

	ident, err := l.Expect("const", lexer.TokIdent)
	if err != nil {
		return nil, err
	}

	if _, err := l.Expect("const", '='); err != nil {
		return nil, err
	}

	val, err := parseValue(s, l)
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

func parseTypedef(s *ast.Specification, l *lexer.Lexer) (*ast.Definition, error) {
	if _, err := l.Expect("typedef", lexer.TokTypedef); err != nil {
		return nil, err
	}

	decl, err := parseDeclaration(s, l)
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

func parseDeclaration(s *ast.Specification, l *lexer.Lexer) (*ast.Declaration, error) {
	var err error
	d := &ast.Declaration{
		Modifier: new(ast.Declaration_Modifier),
	}

	d.Attributes, err = parseAttributes(s, l)
	if err != nil {
		return nil, err
	}

	t := l.Next()
	switch t.ID {
	case lexer.TokUnsigned:
		t, err := l.Expect("declaration", lexer.TokInt, lexer.TokHyper)
		if err != nil {
			return nil, err
		}

		switch t.ID {
		case lexer.TokInt:
			d.Type = ast.UnsignedInt()
		case lexer.TokHyper:
			d.Type = ast.UnsignedHyper()
		}
	case lexer.TokInt:
		d.Type = ast.Int()
	case lexer.TokHyper:
		d.Type = ast.Hyper()
	case lexer.TokFloat:
		d.Type = ast.Float()
	case lexer.TokDouble:
		d.Type = ast.Double()
	case lexer.TokBool:
		d.Type = ast.Bool()
	case lexer.TokString:
		d.Type = ast.String()
	case lexer.TokOpaque:
		d.Type = ast.Opaque()
	case lexer.TokEnum:
		l.Unget(t)
		d.Type, err = parseEnumTypeSpec(s, l)
		if err != nil {
			return nil, err
		}
	case lexer.TokStruct:
		l.Unget(t)
		d.Type, err = parseStructTypeSpec(s, l)
		if err != nil {
			return nil, err
		}
	case lexer.TokUnion:
		l.Unget(t)
		d.Type, err = ParseUnionTypeSpec(s, l)
		if err != nil {
			return nil, err
		}
	case lexer.TokIdent:
		d.Type, err = s.TypeRef(t.Value)
		if err != nil {
			return nil, err
		}
	case lexer.TokVoid:
		d.Type = ast.Void()
		return d, nil
	default:
		return nil, t.Unexpected("declaration")
	}

	if l.NextOneOf('*') != nil {
		d.Modifier.Kind = ast.DECLARATION_MODIFIER_OPTIONAL
	}

	t, err = l.Expect("declaration", lexer.TokIdent)
	if err != nil {
		return nil, err
	}
	d.Name = t.Value

	var t2 *lexer.Token
	if d.Modifier.Kind != ast.DECLARATION_MODIFIER_OPTIONAL {
		t2 = l.NextOneOf('<', '[')
	}

	switch {
	case t2 != nil && t2.ID == '<':
		if l.Peek().ID == '>' {
			d.Modifier.Kind = ast.DECLARATION_MODIFIER_UNBOUNDED
		} else {
			val, err := parseValue(s, l)
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
		val, err := parseValue(s, l)
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

func parseEnumTypeSpec(s *ast.Specification, l *lexer.Lexer) (*ast.Type, error) {
	l.Expect("enum", lexer.TokEnum)
	return parseEnumBody(s, l)
}

func parseEnum(s *ast.Specification, l *lexer.Lexer) (*ast.Definition, error) {
	if _, err := l.Expect("enum", lexer.TokEnum); err != nil {
		return nil, err
	}

	ident, err := l.Expect("enum", lexer.TokIdent)
	if err != nil {
		return nil, err
	}

	// Pre-build a definition slot for this type
	// (This ensures the enum precedes its' associated constants
	// in the output file)
	_, err = s.TypeRef(ident.Value)
	if err != nil {
		return nil, err
	}

	body, err := parseEnumBody(s, l)
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

func parseEnumBody(s *ast.Specification, l *lexer.Lexer) (*ast.Type, error) {
	es := new(ast.EnumSpec)

	if _, err := l.Expect("enum", '{'); err != nil {
		return nil, err
	}

	es.Base = uint32(len(s.Definitions))

body:
	for {
		t, err := l.Expect("enum body", lexer.TokIdent, '[', ',', '}')
		if err != nil {
			return nil, err
		}

		var attributes ast.Attributes
		if t.ID == '[' {
			l.Unget(t)
			attributes, err = parseAttributes(s, l)
			if err != nil {
				return nil, err
			}

			t, err = l.Expect("enum body", lexer.TokIdent)
			if err != nil {
				return nil, err
			}
		}

		if t.ID == lexer.TokIdent {
			if _, err := l.Expect("enum body", '='); err != nil {
				return nil, err
			}
			v, err := parseValue(s, l)
			if err != nil {
				return nil, err
			}
			vu32, err := v.AsU32()
			if err != nil {
				return nil, err
			}

			s.PutDefinition(&ast.Definition{
				Name: t.Value,
				Body: &ast.Definition_Body{
					Kind: ast.DEFINITION_KIND_CONSTANT,
					Constant: &ast.Constant{
						Type:  ast.CONST_ENUM,
						VEnum: vu32,
					},
				},
				Attributes: attributes,
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

	es.Count = uint32(len(s.Definitions)) - es.Base

	return &ast.Type{
		Kind:     ast.TYPE_ENUM,
		EnumSpec: es,
	}, nil
}

func parseStruct(s *ast.Specification, l *lexer.Lexer) (*ast.Definition, error) {
	if _, err := l.Expect("struct", lexer.TokStruct); err != nil {
		return nil, err
	}

	ident, err := l.Expect("struct", lexer.TokIdent)
	if err != nil {
		return nil, err
	}

	// Pre-build a definition slot for this type
	// (This helps the order of our output more closely reflect out input)
	_, err = s.TypeRef(ident.Value)
	if err != nil {
		return nil, err
	}

	body, err := parseStructBody(s, l)
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

func parseStructTypeSpec(s *ast.Specification, l *lexer.Lexer) (*ast.Type, error) {
	if _, err := l.Expect("struct", lexer.TokStruct); err != nil {
		return nil, err
	}

	return parseStructBody(s, l)
}

func parseStructBody(s *ast.Specification, l *lexer.Lexer) (*ast.Type, error) {
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
		decl, err := parseDeclaration(s, l)
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

func ParseUnion(s *ast.Specification, l *lexer.Lexer) (*ast.Definition, error) {
	if _, err := l.Expect("union", lexer.TokUnion); err != nil {
		return nil, err
	}

	ident, err := l.Expect("union", lexer.TokIdent)
	if err != nil {
		return nil, err
	}

	// Pre-build a definition slot for this type
	// (This helps the order of our output more closely reflect out input)
	_, err = s.TypeRef(ident.Value)
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

func ParseUnionTypeSpec(s *ast.Specification, l *lexer.Lexer) (*ast.Type, error) {
	l.Expect("union", lexer.TokUnion)
	return ParseUnionBody(s, l)
}

func ParseUnionBody(s *ast.Specification, l *lexer.Lexer) (*ast.Type, error) {
	var err error
	us := &ast.UnionSpec{
		Options: make(map[uint32]uint32),
	}

	if _, err := l.Expect("union", lexer.TokSwitch); err != nil {
		return nil, err
	}
	if _, err := l.Expect("union", '('); err != nil {
		return nil, err
	}

	us.Discriminant, err = parseDeclaration(s, l)
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
		case lexer.TokDefault, '}':
			break body
		}

		caseTok, err := l.Expect("union", lexer.TokCase)
		if err != nil {
			return nil, err
		}
		discriminantVal, err := parseValue(s, l)
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
		declaration, err := parseDeclaration(s, l)
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

	if caseTok := l.NextOneOf(lexer.TokDefault); caseTok != nil {
		if _, err := l.Expect("union", ':'); err != nil {
			return nil, err
		}
		declaration, err := parseDeclaration(s, l)
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
