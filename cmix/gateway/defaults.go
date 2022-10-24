//go:build !js || !wasm
// +build !js !wasm

// This file is compiled for all architectures except WebAssembly.
package gateway

const (
	MaxPoolSize = 20
)
