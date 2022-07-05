# xx network Client Bindings

## Allowed Types

> At present, only a subset of Go types are supported.
> 
> All exported symbols in the package must have types that are supported. Supported types include:
> 
> - Signed integer and floating point types.
> - String and boolean types.
> - Byte slice types. Note that byte slices are passed by reference, and support mutation.
> - Any function type all of whose parameters and results have supported types. Functions must return either no results, one result, or two results where the type of the second is the built-in 'error' type.
> - Any interface type, all of whose exported methods have supported function types.
> - Any struct type, all of whose exported methods have supported function types and all of whose exported fields have supported types. Unexported symbols have no effect on the cross-language interface, and as such are not restricted.
> 
> The set of supported types will eventually be expanded to cover more Go types, but this is a work in progress.
> 
> Exceptions and panics are not yet supported. If either pass a language boundary, the program will exit.

**Source:** https://pkg.go.dev/golang.org/x/mobile/cmd/gobind under *Type restrictions* heading