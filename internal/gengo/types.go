package gengo

import (
	"fmt"
	"strconv"

	"go.e43.eu/xdrgen/ast"
)

// GoTypeName converts an XDR basic type into its Go equivalent
// Caution: Certain type names (most notably `opaque`) cannot be converted
//
// Returns:
//   * Prefix: '*' Pointer prefix if one needs to be applied to the name of the type
//   * Name: The name of the type
func GoTypeName(
	s *ast.Specification,
	t *ast.Type,
) (
	prefix string,
	name string,
	err error,
) {
	switch t.Kind {
	case ast.TYPE_VOID:
		return "", "struct{}", nil
	case ast.TYPE_BOOL:
		return "", "bool", nil
	case ast.TYPE_INT:
		return "", "int32", nil
	case ast.TYPE_UNSIGNED_INT:
		return "", "uint32", nil
	case ast.TYPE_HYPER:
		return "", "int64", nil
	case ast.TYPE_UNSIGNED_HYPER:
		return "", "uint64", nil
	case ast.TYPE_FLOAT:
		return "", "float32", nil
	case ast.TYPE_DOUBLE:
		return "", "float64", nil
	case ast.TYPE_STRING:
		return "", "string", nil
	case ast.TYPE_REF:
		defn, resolved, err := t.FollowRef(s)
		if err != nil {
			return "", "", err
		}

		switch resolved.Kind {
		case ast.TYPE_STRUCT, ast.TYPE_UNION:
			prefix = "*"
		}
		return prefix, CamelCase(defn.Name), nil
	default:
		return "", "", fmt.Errorf("Don't know how to name a %s", t.Kind)
	}
}

// GoValue converts an ast.Constant into a go literal
func GoValue(v *ast.Constant) string {
	switch v.Type {
	case ast.CONST_VOID:
		return "struct{}{}"
	case ast.CONST_BOOL:
		return fmt.Sprintf("%v", v.VBool)
	case ast.CONST_POS_INT:
		// Heuristic
		if v.VPosInt <= 0xFFFF {
			return fmt.Sprintf("%d", v.VPosInt)
		} else {
			return fmt.Sprintf("0x%X", v.VPosInt)
		}
	case ast.CONST_NEG_INT:
		return fmt.Sprintf("-%d", v.VNegInt)
	case ast.CONST_FLOAT:
		return fmt.Sprintf("%f", v.VFloat)
	case ast.CONST_STRING:
		return strconv.Quote(v.VString)
	case ast.CONST_ENUM:
		return fmt.Sprintf("%d", v.VEnum)
	default:
		panic(fmt.Sprintf("Don't know how to valueize a %s", v.Type))
	}
}
