///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

type Permissioning struct {
	host  *connect.Host
	comms *client.Comms
}

func Init(comms *client.Comms, def *ndf.NetworkDefinition) (*Permissioning, error) {

	perm := Permissioning{
		host:  nil,
		comms: comms,
	}

	var err error
	//add the permissioning host to comms
	hParam := connect.GetDefaultHostParams()
	hParam.AuthEnabled = false

	perm.host, err = comms.AddHost(&id.Permissioning, def.Registration.Address,
		[]byte(def.Registration.TlsCertificate), hParam)

	if err != nil {
		return nil, errors.WithMessage(err, "failed to create permissioning")
	}

	return &perm, nil
}
