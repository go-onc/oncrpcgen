# ONC RPC/XDR Schema Compiler

`oncrpcgen` compiles XDR schemas, as defined in [RFC 4506] and extended in [RFC 5531], 
into a collection of formats:

 * A binary format ([itself defined in XDR][ast]) which can be used to
   implement new compiler backends with relative eas
 * A JSON format derived from the above schema (this can also be used for the
   same purpose, but primarily exists to help inspect the output of the compiler)
 * Go code

This should not yet be considered well tested: this is an early stage project. That said,
it is useful with many XDR schemas.

This implementation stick fairly close to the specifications, though does make some
minor non-standard additions:

 * Attributes can be added to definitions by prefixing them with an attribute set.
   Attribute sets are wrapped in square brackets and formatted as 
   `[a, b, c("string param"), d(1 /* int param)]`. They take the form of a key-value
   map
 * Attributes can be added to the specification itself by ensuring that the first
   non-comment entry in the file is an attribute set preceded by a hash, i.e. 
   `#[foo("bar")]`

Some common attributes are defined:

 * *doc*: A documentation comment for the associated item
 * *mode*: Specifies a non-standard generation mode for a declaration. Currently 
   the only defined mode is `"map"`, which when used on a flexible array declaration
   where the type has two members, will cause a map to be generated in the resulting code
 * *go_package*: Defines what package name to use when generating Go code

[RFC 4506]: https://tools.ietf.org/html/rfc4506 
[RFC 5531]: https://tools.ietf.org/html/rfc5531
[ast]: ast/ast.x