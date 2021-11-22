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
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	receiveMessageTypeErr = "received message is not of type NewFileTransfer"
	protoUnmarshalErr     = "failed to unmarshal request: %+v"
)

// receiveNewFileTransfer starts a thread that waits for new file transfer
// messages.
func (m *Manager) receiveNewFileTransfer(rawMsgs chan message.Receive,
	stop *stoppable.Single) {
	jww.DEBUG.Print("Starting new file transfer message reception thread.")

	for {
		select {
		case <-stop.Quit():
			jww.DEBUG.Print("Stopping new file transfer message reception " +
				"thread: stoppable triggered")
			stop.ToStopped()
			return
		case receivedMsg := <-rawMsgs:
			jww.DEBUG.Print("New file transfer message thread received message.")

			tid, fileName, fileType, sender, size, preview, err :=
				m.readNewFileTransferMessage(receivedMsg)
			if err != nil {
				if err.Error() == receiveMessageTypeErr {
					jww.INFO.Printf("Failed to read message as new file "+
						"transfer message: %+v", err)
				} else {
					jww.WARN.Printf("Failed to read message as new file "+
						"transfer message: %+v", err)
				}
				continue
			}

			// Call the reception callback
			go m.receiveCB(tid, fileName, fileType, sender, size, preview)

			// Trigger a resend of all garbled messages
			m.net.CheckGarbledMessages()
		}
	}
}

// readNewFileTransferMessage reads the received message and adds it to the
// received transfer list. Returns the transfer ID, sender ID, file size, and
// file preview.
func (m *Manager) readNewFileTransferMessage(msg message.Receive) (
	tid ftCrypto.TransferID, fileName, fileType string, sender *id.ID,
	fileSize uint32, preview []byte, err error) {

	// Return an error if the message is not a NewFileTransfer
	if msg.MessageType != message.NewFileTransfer {
		err = errors.New(receiveMessageTypeErr)
		return
	}

	// Unmarshal the request message
	newFT := &NewFileTransfer{}
	err = proto.Unmarshal(msg.Payload, newFT)
	if err != nil {
		err = errors.Errorf(protoUnmarshalErr, err)
		return
	}

	// Get RNG from stream
	rng := m.rng.GetStream()
	defer rng.Close()

	// Add the transfer to the list of receiving transfers
	key := ftCrypto.UnmarshalTransferKey(newFT.TransferKey)
	numParts := uint16(newFT.NumParts)
	numFps := calcNumberOfFingerprints(numParts, newFT.Retry)
	tid, err = m.received.AddTransfer(
		key, newFT.TransferMac, newFT.Size, numParts, numFps, rng)
	if err != nil {
		return
	}

	return tid, newFT.FileName, newFT.FileType, msg.Sender, newFT.Size,
		newFT.Preview, nil
}
