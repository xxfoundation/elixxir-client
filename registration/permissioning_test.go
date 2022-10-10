////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package registration

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelTrace)
	connect.TestingOnlyDisableTLS = true
	os.Exit(m.Run())
}

// Init should create a valid Registration communications struct
func TestInit(t *testing.T) {
	// Create dummy comms and ndf
	comms, err := client.NewClientComms(id.NewIdFromUInt(100, id.User, t), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	def := &ndf.NetworkDefinition{
		Registration: ndf.Registration{
			EllipticPubKey: "MqaJJ3GjFisNRM6LRedRnooi14gepMaQxyWctXVU",
		},
	}
	reg, err := Init(comms, def)
	if err != nil {
		t.Fatal(err)
	}
	if reg.comms == nil {
		t.Error("reg comms returned should not be nil")
	}
	if reg.host == nil {
		t.Error("reg host returned should not be nil")
	}
}
