# Client Bindings

"bindings" is the client bindings which can be used to generate Android
and iOS client libraries for apps using gomobile. Gomobile is
limited to int, string, []byte, interfaces, and only a couple other types, so
it is necessary to define several interfaces to support passing more complex
data across the boundary (see `interfaces.go`). The rest of the logic
is located in `api.go`
