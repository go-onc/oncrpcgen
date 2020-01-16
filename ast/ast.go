package ast

import (
	"errors"
	"fmt"
)

//go:generate go run ../. -Gxb,go ast.x

var (
	ErrDefinitionNotFound      = errors.New("Definition not found")
	ErrDefinitionNotType       = errors.New("Definition not a type")
	ErrDefinitionNotConstant   = errors.New("Definition not a constant")
	ErrDefinitionNotConsistent = errors.New("Definition not consistent with preceding definition")
	ErrRedefinitionOfType      = errors.New("Attempt to redefine type")
	ErrRedefinitionOfConstant  = errors.New("Attempt to redefine constant")
	ErrTypeNotRef              = errors.New("Type is not of TYPE_REF")
)

// Void returns a type of TYPE_VOID
func Void() *Type { return &Type{Kind: TYPE_VOID} }

// Bool returns a type of TYPE_BOOL
func Bool() *Type { return &Type{Kind: TYPE_BOOL} }

// Int returns a type of TYPE_INT
func Int() *Type { return &Type{Kind: TYPE_INT} }

// UnsignedInt returns a type of TYPE_UNSIGNED_INT
func UnsignedInt() *Type { return &Type{Kind: TYPE_UNSIGNED_INT} }

// Hyper returns a type of TYPE_HYPER
func Hyper() *Type { return &Type{Kind: TYPE_HYPER} }

// UnsignedHyper returns a type of TYPE_UNSIGNED_HYPER
func UnsignedHyper() *Type { return &Type{Kind: TYPE_UNSIGNED_HYPER} }

// Float returns a type of TYPE_FLOAT
func Float() *Type { return &Type{Kind: TYPE_FLOAT} }

// Double returns a type of TYPE_DOUBLE
func Double() *Type { return &Type{Kind: TYPE_DOUBLE} }

// String returns a type of TYPE_STRING
func String() *Type { return &Type{Kind: TYPE_STRING} }

// Opaque returns a type of TYPE_OPAQUE
func Opaque() *Type { return &Type{Kind: TYPE_OPAQUE} }

// Ref returns a type which is a reference to the specified name
func Ref(id uint32) *Type {
	return &Type{
		Kind: TYPE_REF,
		Ref:  id,
	}
}

// NamedDefinition looks for a definition with the specified name
func (s *Specification) NamedDefinition(n string) *Definition {
	for _, d := range s.Definitions {
		if d.Name == n {
			return d
		}
	}
	return nil
}

// PutDefinition appends a definition with the speicifed name, if it
// would not conflict with one which already exists
func (s *Specification) PutDefinition(d *Definition) (uint32, error) {
	var (
		xdIdx uint32
		xd    *Definition
	)
	for i, x := range s.Definitions {
		if xd.Name == d.Name {
			xdIdx = uint32(i)
			xd = x
			break
		}
	}

	if xd != nil {
		if xd.Body.Kind != d.Body.Kind {
			return 0, ErrDefinitionNotConsistent
		} else if xd.Body.Kind == DEFINITION_KIND_TYPE && xd.Body.Type != nil {
			return 0, ErrRedefinitionOfType
		} else if xd.Body.Kind == DEFINITION_KIND_CONSTANT && xd.Body.Constant != nil {
			return 0, ErrRedefinitionOfConstant
		}
		s.Definitions[xdIdx] = d
		return xdIdx, nil
	}

	s.Definitions = append(s.Definitions, d)
	return uint32(len(s.Definitions) - 1), nil
}

// TypeRef ensures a (potentially empty) type definition exists with the specified
// name
func (s *Specification) TypeRef(name string) (*Type, error) {
	for i, xd := range s.Definitions {
		if xd.Name == name {
			if xd.Body.Kind != DEFINITION_KIND_TYPE {
				return nil, ErrDefinitionNotType
			}
			return Ref(uint32(i)), nil
		}
	}

	d := &Definition{
		Name: name,
		Body: &Definition_Body{
			Kind: DEFINITION_KIND_TYPE,
		},
	}
	s.Definitions = append(s.Definitions, d)
	return Ref(uint32(len(s.Definitions) - 1)), nil
}

// GetType looks up the named type, returning an error if it is not found
func (s *Specification) GetType(n string) (*Type, error) {
	d := s.NamedDefinition(n)
	if d == nil {
		return nil, ErrDefinitionNotFound
	}

	if d.Body.Kind != DEFINITION_KIND_TYPE {
		return nil, ErrDefinitionNotType
	}

	return d.Body.Type, nil
}

// GetConstant looks up the named constant, returning an error if it is not found
func (s *Specification) GetConstant(n string) (*Constant, error) {
	d := s.NamedDefinition(n)
	if d == nil {
		return nil, ErrDefinitionNotFound
	}

	if d.Body.Kind != DEFINITION_KIND_CONSTANT {
		return nil, ErrDefinitionNotConstant
	}

	return d.Body.Constant, nil
}

// HasMember returns if a named member exists
func (ss *StructSpec) HasMember(name string) bool {
	for _, m := range ss.Members {
		if m.Name == name {
			return true
		}
	}
	return false
}

// HasOption returns if a named option exists
func (es *EnumSpec) HasOption(name string) bool {
	for _, o := range es.Options {
		if o.Name == name {
			return true
		}
	}
	return false
}

// GetName returns the canonical (first) name for the specified numeric value
func (es *EnumSpec) GetName(val uint32) string {
	for _, o := range es.Options {
		if o.Value == val {
			return o.Name
		}
	}
	return ""
}

// HasOption returns if the numeric value specified is defined
func (us *UnionSpec) HasOption(val uint32) bool {
	_, exists := us.Options[val]
	return exists
}

// HasMember returns if a member is already defined with the specified name
func (us *UnionSpec) HasMember(name string) bool {
	for _, m := range us.Members {
		if name == m.Name {
			return true
		}
	}
	return false
}

// GetMember returns the member with the specified name, and its index, if it exists
func (us *UnionSpec) GetMember(name string) (uint32, *Declaration) {
	for i, m := range us.Members {
		if name == m.Name {
			return uint32(i), m
		}
	}
	return 0, nil
}

// AsU32 attempts to reinterpret a constant as an unsigned 32-bit number
func (c *Constant) AsU32() (uint32, error) {
	if c.Type == CONST_POS_INT {
		return uint32(c.VPosInt), nil
	} else if c.Type == CONST_NEG_INT {
		return uint32(c.VNegInt), nil
	} else if c.Type == CONST_ENUM {
		return c.VEnum, nil
	} else {
		return 0, fmt.Errorf("Can't use constant %s as integer", c.Type)
	}
}

// IsVoid returns if this is a void (empty) declaration
func (d *Declaration) IsVoid() bool {
	return d.Type.Kind == TYPE_VOID
}

// Equal returns if two declarations are equal
func (l *Declaration) Equal(r *Declaration) bool {
	return *l.Type == *r.Type && *l.Modifier == *r.Modifier && l.Name == r.Name
}

// FollowRef follows a single level of TYPE_REF, returning the
// underlying type
func (t *Type) FollowRef(s *Specification) (*Definition, *Type, error) {
	if t.Kind != TYPE_REF {
		return nil, nil, ErrTypeNotRef
	}

	if uint(t.Ref) >= uint(len(s.Definitions)) {
		return nil, nil, ErrDefinitionNotFound
	}

	d := s.Definitions[t.Ref]
	if d.Body.Kind != DEFINITION_KIND_TYPE {
		return nil, nil, ErrDefinitionNotType
	}

	return d, d.Body.Type, nil
}

// Resolve attempts to resolve a TYPE_REF into the underlying type, by looking at
// the passed specification
func (t *Type) Resolve(s *Specification) (*Type, error) {
	var err error
	for {
		if t.Kind != TYPE_REF {
			return t, nil
		}

		_, t, err = t.FollowRef(s)
		if err != nil {
			return nil, err
		}
	}
}

// GetStringDefault attempts to look up the named attribute as a string,
// or returns the specified default
func (as Attributes) GetStringDefault(name, def string) string {
	if v, ok := as[name]; ok && v.Type == CONST_STRING {
		return v.VString
	} else {
		return def
	}
}

// GetString attempts to look up the named attribute as a string, or returns
// the empty string
func (as Attributes) GetString(name string) string {
	return as.GetStringDefault(name, "")
}
