////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

//go:build !js || !wasm

// This file is compiled for all architectures except WebAssembly.
package gateway

import (
	"gitlab.com/xx_network/primitives/id"
)

const (
	MaxPoolSize = 20
)

// getAddress returns the correct connection info. For non webassembly,
// it is a simple pass through. For webassembly, it constructs the
// gateway url and returns a nil cert
func getConnectionInfo(gwId *id.ID, gwAddr, certificate string) (addr string, cert []byte, err error) {
	addr = gwAddr
	cert = []byte(certificate)

	return addr, cert, nil
}
