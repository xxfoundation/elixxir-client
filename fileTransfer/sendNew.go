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
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	newFtProtoMarshalErr = "failed to form new file transfer message: %+v"
	newFtSendE2eErr      = "failed to send new file transfer message via E2E to recipient %s: %+v"
)

// sendNewFileTransfer sends the initial file transfer message over E2E.
func (m *Manager) sendNewFileTransfer(recipient *id.ID, fileName,
	fileType string, key ftCrypto.TransferKey, mac []byte, numParts uint16,
	fileSize uint32, retry float32, preview []byte) error {

	// Create Send message with marshalled NewFileTransfer
	sendMsg, err := newNewFileTransferE2eMessage(recipient, fileName, fileType,
		key, mac, numParts, fileSize, retry, preview)
	if err != nil {
		return errors.Errorf(newFtProtoMarshalErr, err)
	}

	// get partner relationship so that the silent preimage can be generated
	relationship, err := m.store.E2e().GetPartner(recipient)
	if err != nil {
		return err
	}

	// Sends as a silent message to avoid a notification
	p := params.GetDefaultE2E()
	p.CMIX.IdentityPreimage = relationship.GetSilentPreimage()
	p.DebugTag = "ft.New"

	// Send E2E message
	rounds, _, _, err := m.net.SendE2E(sendMsg, p, nil)
	if err != nil && len(rounds) == 0 {
		return errors.Errorf(newFtSendE2eErr, recipient, err)
	}

	return nil
}

// newNewFileTransferE2eMessage generates the message.Send for the given
// recipient containing the marshalled NewFileTransfer message.
func newNewFileTransferE2eMessage(recipient *id.ID, fileName, fileType string,
	key ftCrypto.TransferKey, mac []byte, numParts uint16, fileSize uint32,
	retry float32, preview []byte) (message.Send, error) {

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
	marshalledMsg, err := proto.Marshal(protoMsg)
	if err != nil {
		return message.Send{}, err
	}

	// Create message.Send of the type NewFileTransfer
	sendMsg := message.Send{
		Recipient:   recipient,
		Payload:     marshalledMsg,
		MessageType: message.NewFileTransfer,
	}

	return sendMsg, nil
}
