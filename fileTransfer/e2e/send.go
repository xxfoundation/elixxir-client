////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e"
	ft "gitlab.com/elixxir/client/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	// sendNewFileTransferMessage
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
	recipient *id.ID, transferInfo []byte, e2eHandler e2eHandler) error {

	// Get E2E parameters
	params := e2e.GetDefaultParams()
	params.ServiceTag = catalog.Silent
	params.LastServiceTag = catalog.Silent
	params.DebugTag = initialMessageDebugTag

	_, err := e2eHandler.SendE2E(
		catalog.NewFileTransfer, recipient, transferInfo, params)
	if err != nil {
		return errors.Errorf(errNewFtSendE2e, err)
	}

	return nil
}

// sendEndFileTransferMessage sends an E2E message to the recipient informing
// them that all file parts have arrived once the network is healthy.
func sendEndFileTransferMessage(
	recipient *id.ID, cmix ft.Cmix, e2eHandler e2eHandler) {
	callbackID := make(chan uint64, 1)
	callbackID <- cmix.AddHealthCallback(
		func(healthy bool) {
			if healthy {
				params := e2e.GetDefaultParams()
				params.LastServiceTag = catalog.EndFT
				params.DebugTag = lastMessageDebugTag

				_, err := e2eHandler.SendE2E(
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
