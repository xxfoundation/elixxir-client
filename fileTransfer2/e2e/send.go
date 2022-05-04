////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e"
	ft "gitlab.com/elixxir/client/fileTransfer2"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	// sendNewFileTransferMessage
	errMarshalInfo  = "failed to marshal new transfer info: %+v"
	errNewFtSendE2e = "failed to send initial file transfer message via E2E: %+v"

	// sendEndFileTransferMessage
	errEndFtSendE2e = "[FT] Failed to send ending file transfer message via E2E: %+v"
)

const (
	// Tag that is used for log printing in SendE2E when sending the initial
	// message
	initialMessageDebugTag = "FT.New"

	// Tag that is used for log printing in SendE2E when sending the ending
	// message
	lastMessageDebugTag = "FT.End"
)

// sendNewFileTransferMessage sends an E2E message to the recipient informing
// them of the incoming file transfer.
func sendNewFileTransferMessage(
	recipient *id.ID, info *ft.TransferInfo, e2eHandler E2e) error {

	// Marshal the message
	payload, err := info.Marshal()
	if err != nil {
		return errors.Errorf(errMarshalInfo, err)
	}

	// Get E2E parameters
	params := e2e.GetDefaultParams()
	params.ServiceTag = catalog.Silent
	params.LastServiceTag = catalog.Silent
	params.DebugTag = initialMessageDebugTag

	_, _, _, err = e2eHandler.SendE2E(
		catalog.NewFileTransfer, recipient, payload, params)
	if err != nil {
		return errors.Errorf(errNewFtSendE2e, err)
	}

	return nil
}

// sendEndFileTransferMessage sends an E2E message to the recipient informing
// them that all file parts have arrived once the network is healthy.
func sendEndFileTransferMessage(recipient *id.ID, cmix ft.Cmix, e2eHandler E2e) {
	callbackID := make(chan uint64, 1)
	callbackID <- cmix.AddHealthCallback(
		func(healthy bool) {
			if healthy {
				params := e2e.GetDefaultParams()
				params.LastServiceTag = catalog.EndFT
				params.DebugTag = lastMessageDebugTag

				_, _, _, err := e2eHandler.SendE2E(
					catalog.EndFileTransfer, recipient, nil, params)
				if err != nil {
					jww.ERROR.Printf(errEndFtSendE2e, err)
				}

				cbID := <-callbackID
				cmix.RemoveHealthCallback(cbID)
			}
		},
	)
}
