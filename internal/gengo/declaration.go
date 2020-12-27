package gengo

import (
	"fmt"
	"strings"

	"go.e43.eu/xdrgen/ast"
)

type declMode int

const (
	declModeDefault declMode = iota
	declModeUnionSwitch
	declModeUnionOption
	declModeUnionDefault
	declModeTypedef
)

func checkMapLikeDef(s *ast.Specification, d *ast.Declaration) error {
	t, err := d.Type.Resolve(s)
	if err != nil {
		return err
	}
	if t.Kind != ast.TYPE_STRUCT {
		return fmt.Errorf("Can't generate %s as map because type must be a struct, is %s", d.Name, t.Kind)
	}

	if len(t.StructSpec.Members) != 2 {
		return fmt.Errorf("Can't generate %s as map becasue struct has %d members, must be 2", d.Name, len(t.StructSpec.Members))
	}

	if d.Modifier.Kind != ast.DECLARATION_MODIFIER_FLEXIBLE && d.Modifier.Kind != ast.DECLARATION_MODIFIER_UNBOUNDED {
		return fmt.Errorf("Can't generate %s as map because modifier type %s unsupported", d.Name, d.Modifier.Kind)
	}
	return nil
}

// Generates a basic (not union or typedef) declaration
func GenBasicDeclaration(spec *ast.Specification, d *ast.Declaration) (string, error) {
	decl, _, err := genDeclarationCore(spec, d, declModeDefault, nil)
	return decl, err
}

// Generates a typedef declaration
// Returns the tags that would be applied (so that a proxy marshaller may be generated if
// appropriate)
func GenTypedefDeclaration(
	spec *ast.Specification,
	d *ast.Declaration,
) (
	decl string,
	tags []string,
	err error,
) {
	return genDeclarationCore(spec, d, declModeTypedef, nil)
}

// Generates a union switch declaration
func GenUnionSwitchDeclaration(spec *ast.Specification, d *ast.Declaration) (string, error) {
	decl, _, err := genDeclarationCore(spec, d, declModeUnionSwitch, nil)
	return decl, err
}

type UnionDeclaration struct {
	decl      *ast.Declaration
	variants  []uint32
	isDefault bool
}

// Generate a declaration for a union variant
func GenUnionDeclaration(spec *ast.Specification, d *UnionDeclaration) (string, error) {
	mode := declModeUnionOption
	if d.isDefault {
		mode = declModeUnionDefault
	}
	decl, _, err := genDeclarationCore(spec, d.decl, mode, d.variants)
	return decl, err
}

func genDeclarationCore(
	spec *ast.Specification,
	d *ast.Declaration,
	mode declMode,
	variants []uint32,
) (string, []string, error) {
	var (
		s    string
		tags []string
	)

	switch mode {
	case declModeUnionSwitch:
		tags = append(tags, "union:switch")
	case declModeUnionOption:
		variantStrs := make([]string, len(variants))
		for i, v := range variants {
			variantStrs[i] = fmt.Sprintf("%d", v)
		}
		tags = append(tags, fmt.Sprintf("union:%s", strings.Join(variantStrs, ",")))
	case declModeUnionDefault:
		tags = append(tags, fmt.Sprintf("union:default"))
	}

	if d.Attributes.GetString("mode") == "map" {
		if err := checkMapLikeDef(spec, d); err != nil {
			return "", nil, err
		}

		t, err := d.Type.Resolve(spec)
		if err != nil {
			return "", nil, err
		}

		keyPfx, keyType, err := GoTypeName(spec, t.StructSpec.Members[0].Type)
		if err != nil {
			return "", nil, err
		}

		if keyPfx != "" {
			return "", nil, fmt.Errorf("Can't use %s as map key", keyType)
		}

		valPfx, valType, err := GoTypeName(spec, t.StructSpec.Members[1].Type)
		if err != nil {
			return "", nil, err
		}

		s = fmt.Sprintf("%s map[%s]%s%s", CamelCase(d.Name), keyType, valPfx, valType)
	} else if d.Type.Kind == ast.TYPE_STRING {
		s = fmt.Sprintf("%s string", CamelCase(d.Name))
		switch d.Modifier.Kind {
		case ast.DECLARATION_MODIFIER_NONE, ast.DECLARATION_MODIFIER_OPTIONAL:
			return "", nil, fmt.Errorf("Non-array string")
		case ast.DECLARATION_MODIFIER_FIXED:
			tags = append(tags, fmt.Sprintf("len:%d", d.Modifier.Size))
		case ast.DECLARATION_MODIFIER_FLEXIBLE:
			tags = append(tags, fmt.Sprintf("maxlen:%d", d.Modifier.Size))
		}
	} else if d.Type.Kind == ast.TYPE_OPAQUE {
		switch d.Modifier.Kind {
		case ast.DECLARATION_MODIFIER_NONE, ast.DECLARATION_MODIFIER_OPTIONAL:
			return "", nil, fmt.Errorf("Non-array opaque")
		case ast.DECLARATION_MODIFIER_FIXED:
			s = fmt.Sprintf("%s [%d]byte", CamelCase(d.Name), d.Modifier.Size)
		case ast.DECLARATION_MODIFIER_FLEXIBLE:
			s = fmt.Sprintf("%s []byte", CamelCase(d.Name))
			tags = append(tags, fmt.Sprintf("maxlen:%d", d.Modifier.Size))
		case ast.DECLARATION_MODIFIER_UNBOUNDED:
			s = fmt.Sprintf("%s []byte", CamelCase(d.Name))
		}
		tags = append(tags, "opaque")
	} else {
		pfx, typeName, err := GoTypeName(spec, d.Type)
		if err != nil {
			return "", nil, err
		}
		switch d.Modifier.Kind {
		case ast.DECLARATION_MODIFIER_NONE:
			// Nothing
		case ast.DECLARATION_MODIFIER_OPTIONAL:
			// pfx might already be *. That's OK
			tags = append(tags, "opt")
			pfx = "*"
		case ast.DECLARATION_MODIFIER_FIXED:
			pfx = fmt.Sprintf("[%d]%s", d.Modifier.Size, pfx)
		case ast.DECLARATION_MODIFIER_FLEXIBLE:
			pfx = "[]" + pfx
		case ast.DECLARATION_MODIFIER_UNBOUNDED:
			pfx = "[]" + pfx
		}
		s = fmt.Sprintf("%s %s%s", CamelCase(d.Name), pfx, typeName)
	}

	if mode != declModeTypedef {
		omitEmpty := ""
		if (mode == declModeUnionOption) != (d.Modifier.Kind == ast.DECLARATION_MODIFIER_OPTIONAL) {
			omitEmpty = ",omitempty"
		}
		xdrt := ""
		if len(tags) > 0 {
			xdrt = fmt.Sprintf("xdr:\"%s\" ", strings.Join(tags, "/"))
		}
		s = fmt.Sprintf("%s `%sjson:\"%s%s\"`", s, xdrt, d.Name, omitEmpty)
	}

	comment := DocComment(d.Attributes, "")
	if comment != "" {
		s = fmt.Sprintf("%s\n%s", comment, s)
	}

	return s, tags, nil
}
