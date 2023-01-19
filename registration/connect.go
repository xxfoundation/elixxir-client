//go:build !js || !wasm
// +build !js !wasm

// This file is compiled for all architectures except WebAssembly.
package registration

// getAddress returns the correct connection info. For non webassembly,
// it is a simple pass through. For webassembly, it does not
// return the cert
func getConnectionInfo(regAddr, certificate string) (addr string, cert []byte, err error) {
	addr = regAddr
	cert = []byte(certificate)

	return addr, cert, nil
}
