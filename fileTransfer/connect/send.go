////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/catalog"
	"gitlab.com/elixxir/client/v5/e2e"
	ft "gitlab.com/elixxir/client/v5/fileTransfer"
)

// Error messages.
const (
	// sendNewFileTransferMessage
	errNewFtSendE2e = "failed to send initial file transfer message via connection E2E: %+v"

	// sendEndFileTransferMessage
	errEndFtSendE2e = "[FT] Failed to send ending file transfer message via connection E2E: %+v"
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
	transferInfo []byte, connectionHandler connection) error {

	// Get E2E parameters
	params := e2e.GetDefaultParams()
	params.ServiceTag = catalog.Silent
	params.LastServiceTag = catalog.Silent
	params.DebugTag = initialMessageDebugTag

	_, err := connectionHandler.SendE2E(
		catalog.NewFileTransfer, transferInfo, params)
	if err != nil {
		return errors.Errorf(errNewFtSendE2e, err)
	}

	return nil
}

// sendEndFileTransferMessage sends an E2E message to the recipient informing
// them that all file parts have arrived once the network is healthy.
func sendEndFileTransferMessage(cmix ft.Cmix, connectionHandler connection) {
	callbackID := make(chan uint64, 1)
	callbackID <- cmix.AddHealthCallback(
		func(healthy bool) {
			if healthy {
				params := e2e.GetDefaultParams()
				params.LastServiceTag = catalog.EndFT
				params.DebugTag = lastMessageDebugTag

				_, err := connectionHandler.SendE2E(
					catalog.EndFileTransfer, nil, params)
				if err != nil {
					jww.ERROR.Printf(errEndFtSendE2e, err)
				}

				cbID := <-callbackID
				cmix.RemoveHealthCallback(cbID)
			}
		},
	)
}
