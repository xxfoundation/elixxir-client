package io

import (
	"crypto/sha256"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/ndf"
)

var noNDFErr = errors.New("Failed to get ndf from permissioning: rpc error: code = Unknown desc = Permissioning server does not have an ndf to give to client")

//GetUpdatedNDF: Connects to the permissioning server to get the updated NDF from it
func PollNdf(currentDef *ndf.NetworkDefinition, comms *client.Comms) (*ndf.NetworkDefinition, error) {
	//Hash the client's ndf for comparison with registration's ndf
	hash := sha256.New()
	ndfBytes := currentDef.Serialize()
	hash.Write(ndfBytes)
	ndfHash := hash.Sum(nil)

	//Put the hash in a message
	msg := &mixmessages.NDFHash{Hash: ndfHash}

	host, ok := comms.GetHost(PermissioningAddrID)
	if !ok {
		return nil, errors.New("Failed to find permissioning host")
	}

	//Send the hash to registration
	response, err := comms.RequestNdf(host, msg)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get ndf from permissioning: %v", err)
		if errMsg == noNDFErr.Error() {
			globals.Log.WARN.Println("Continuing without an updated NDF")
			return nil, nil
		}
		return nil, errors.New(errMsg)
	}

	//If there was no error and the response is nil, client's ndf is up-to-date
	if response == nil || response.Ndf == nil {
		globals.Log.DEBUG.Printf("Client NDF up-to-date")
		return nil, nil
	}

	//FixMe: use verify instead? Probs need to add a signature to ndf, like in registration's getupdate?

	globals.Log.INFO.Printf("Remote NDF: %s", string(response.Ndf))

	//Otherwise pull the ndf out of the response
	updatedNdf, _, err := ndf.DecodeNDF(string(response.Ndf))
	if err != nil {
		//If there was an error decoding ndf
		errMsg := fmt.Sprintf("Failed to decode response to ndf: %v", err)
		return nil, errors.New(errMsg)
	}
	return updatedNdf, nil
}
