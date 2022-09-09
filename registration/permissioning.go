////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package registration

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"math"
	"time"
)

type Registration struct {
	host  *connect.Host
	comms *client.Comms
}

func Init(comms *client.Comms, def *ndf.NetworkDefinition) (*Registration, error) {

	perm := Registration{
		host:  nil,
		comms: comms,
	}

	var err error
	//add the registration host to comms
	hParam := connect.GetDefaultHostParams()
	hParam.AuthEnabled = false
	// Do not send KeepAlive packets
	hParam.KaClientOpts.Time = time.Duration(math.MaxInt64)
	hParam.MaxRetries = 3
	perm.host, err = comms.AddHost(&id.ClientRegistration, def.Registration.ClientRegistrationAddress,
		[]byte(def.Registration.TlsCertificate), hParam)

	if err != nil {
		return nil, errors.WithMessage(err, "failed to create registration")
	}

	_, err = comms.AddHost(&id.Permissioning, def.Registration.Address, // We need to add this for round updates to work
		[]byte(def.Registration.TlsCertificate), hParam)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create permissioning")
	}

	return &perm, nil
}
