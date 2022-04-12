////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	// manager.sendNewFileTransferMessage
	errProtoMarshal = "failed to proto marshal NewFileTransfer: %+v"
	errNewFtSendE2e = "failed to send initial file transfer message via E2E: %+v"

	// manager.sendEndFileTransferMessage
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
func (m *manager) sendNewFileTransferMessage(recipient *id.ID, fileName,
	fileType string, key *ftCrypto.TransferKey, mac []byte, numParts uint16,
	fileSize uint32, retry float32, preview []byte) error {

	// Construct NewFileTransfer message
	protoMsg := &NewFileTransfer{
		FileName:    fileName,
		FileType:    fileType,
		TransferKey: key.Bytes(),
		TransferMac: mac,
		NumParts:    uint32(numParts),
		Size:        fileSize,
		Retry:       retry,
		Preview:     preview,
	}

	// Marshal the message
	payload, err := proto.Marshal(protoMsg)
	if err != nil {
		return errors.Errorf(errProtoMarshal, err)
	}

	// Get E2E parameters
	params := e2e.GetDefaultParams()
	params.ServiceTag = catalog.Silent
	params.LastServiceTag = catalog.Silent
	params.CMIX.DebugTag = initialMessageDebugTag

	_, _, _, err = m.e2e.SendE2E(
		catalog.NewFileTransfer, recipient, payload, params)
	if err != nil {
		return errors.Errorf(errNewFtSendE2e, err)
	}

	return nil
}

// sendEndFileTransferMessage sends an E2E message to the recipient informing
// them that all file parts have arrived once the network is healthy.
func (m *manager) sendEndFileTransferMessage(recipient *id.ID) {
	callbackID := make(chan uint64, 1)
	callbackID <- m.cmix.AddHealthCallback(
		func(healthy bool) {
			if healthy {
				params := e2e.GetDefaultParams()
				params.LastServiceTag = catalog.EndFT
				params.CMIX.DebugTag = lastMessageDebugTag

				_, _, _, err := m.e2e.SendE2E(
					catalog.EndFileTransfer, recipient, nil, params)
				if err != nil {
					jww.ERROR.Printf(errEndFtSendE2e, err)
				}

				cbID := <-callbackID
				m.cmix.RemoveHealthCallback(cbID)
			}
		},
	)
}
