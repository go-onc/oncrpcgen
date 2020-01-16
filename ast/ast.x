#[
	doc("XDR Abstract Syntax Tree definition - a binary interchange format for XDR specifications"),
	go_package("ast"),
]

[doc("Binary magic: the `magic` field of the `specification` should be set to this value")]
const XDR_BIN_MAGIC = 0x895844520D0A1A0A;

[doc("Root object of a specification")]
struct specification {
	[doc("Magic number: set to XDR_BIN_MAGIC")]
	unsigned hyper magic;
	
	[doc("Spec attributes (set using pragma directives)")]
	attributes attributes;

	[doc("List of all definitions")]
	definition definitions<>;
};

[doc("An attribute of an object")]
struct attribute {
	string   name<>;
	constant value;
};

[doc("A set of attributes")]
typedef [mode("map")] attribute attributes<>;

[doc("The kind of a definition")]
enum definition_kind {
	DEFINITION_KIND_TYPE  = 0,
	DEFINITION_KIND_CONSTANT = 1,
};

[doc("A top-level definition")]
struct definition {
	[doc("The name of the definition")]
	string name<>;
	[doc("The attributes of the definition")]
	attributes attributes;

	union switch(definition_kind kind) {
	case DEFINITION_KIND_TYPE:
		[doc("Body, for type definitions")]
		type type;
	case DEFINITION_KIND_CONSTANT:
		[doc("Body, for constant definitions")]
		constant constant;
	} body;
};

[doc("The kind of the type")]
enum type_kind {
	TYPE_VOID = 0,
	TYPE_BOOL = 1,
	TYPE_INT  = 2,
	TYPE_UNSIGNED_INT = 3,
	TYPE_HYPER = 4,
	TYPE_UNSIGNED_HYPER = 5,
	TYPE_FLOAT = 6,
	TYPE_DOUBLE = 7, 
	TYPE_STRING = 8,
	TYPE_OPAQUE = 9,
	TYPE_ENUM = 10,
	TYPE_STRUCT = 11,
	TYPE_UNION = 12,
	TYPE_REF = 13,
	TYPE_TYPEDEF = 14,
};

[doc("Definition of a type")]
union type switch(type_kind kind) {
case TYPE_VOID:           void;
case TYPE_BOOL:           void;
case TYPE_INT:            void;
case TYPE_UNSIGNED_INT:   void;
case TYPE_HYPER:          void;
case TYPE_UNSIGNED_HYPER: void;
case TYPE_FLOAT:          void;
case TYPE_DOUBLE:         void;
case TYPE_STRING:         void;
case TYPE_OPAQUE:         void;
case TYPE_ENUM:           enum_spec enum_spec;
case TYPE_STRUCT:         struct_spec struct_spec;
case TYPE_UNION:          union_spec union_spec;
case TYPE_REF:            unsigned int ref;
case TYPE_TYPEDEF:        declaration type_def;
};

[doc("Definition of an enum")]
struct enum_spec {
	// We don't use a Map here to preserve ordering
	[doc("Set of all options")]
	struct {
		[doc("Name of the enumerant")]
		string   name<>;
		[doc("Value of the enumerant")]
		unsigned int value;
	} options<>;
};

[doc("How a declaration modifies its type")]
enum declaration_modifier {
	DECLARATION_MODIFIER_NONE      = 0,
	DECLARATION_MODIFIER_OPTIONAL  = 1,
	DECLARATION_MODIFIER_FIXED     = 2,
	DECLARATION_MODIFIER_FLEXIBLE  = 3,
	DECLARATION_MODIFIER_UNBOUNDED = 5,
};

[doc("Field declaration")]
struct declaration {
	[doc("Type of the field")]
	type   type;
	[doc("Name of the field")]
	string name<>;
	[doc("Modifier of the type")]
	union switch (declaration_modifier kind) {
		case DECLARATION_MODIFIER_NONE:      void;
		case DECLARATION_MODIFIER_OPTIONAL:  void;
		case DECLARATION_MODIFIER_FIXED:     unsigned int size;
		case DECLARATION_MODIFIER_FLEXIBLE:  unsigned int size;
		case DECLARATION_MODIFIER_UNBOUNDED: void;
	} modifier;
	[doc("Field attributes")]
	attributes attributes;
};

[doc("Definition of an enum")]
struct struct_spec {
	[doc("Set of struct members")]
	declaration members<>;
};

[doc("Definition of a union")]
struct union_spec {
	[doc("Discriminant field")]
	declaration discriminant;
	[doc("Set of union member fields")]
	declaration members<>;
	[doc("Mapping from values to union member. `member` is the index of the member in `members`"), mode("map")]
	struct {
		unsigned int value;
		unsigned int member;
	} options<>;
	[doc("If a default member is present, defines it")]
	unsigned int *default_member;
};

[doc("Type of a constant. These are a subset of XDR types")]
enum constant_kind {
	// [doc("Boolean")]
	CONST_BOOL    = 0,
	// [doc("Positive integer (unsigned hyper)")]
	CONST_POS_INT = 1,
	// [doc("Negative integer (negate as a signed hyper)")]
	CONST_NEG_INT = 2,
	// [doc("Double precision floating point value")]
	CONST_FLOAT   = 3,
	// [doc("String constant")]
	CONST_STRING  = 4,
	// [doc("Enumeration value")]
	CONST_ENUM    = 5,
};

union constant switch(constant_kind type) {
case CONST_BOOL:    bool           v_bool;
case CONST_POS_INT: unsigned hyper v_pos_int;
case CONST_NEG_INT: unsigned hyper v_neg_int;
case CONST_FLOAT:   double         v_float;
case CONST_STRING:  string         v_string<>;
case CONST_ENUM:    unsigned int   v_enum;
};