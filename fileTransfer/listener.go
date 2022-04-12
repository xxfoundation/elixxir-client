////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/fileTransfer/store"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
)

// Error messages.
const (
	errProtoUnmarshal  = "[FT] Failed to proto unmarshal new file transfer request: %+v"
	errNewRtTransferID = "[FT] Failed to generate transfer ID for new received file transfer: %+v"
	errAddNewRt        = "[FT] Failed to add new file transfer %s (%q): %+v"
)

// Name of listener (used for debugging)
const listenerName = "NewFileTransferListener"

// fileTransferListener waits for new file transfer messages to get ready to
// receive the file transfer parts. Adheres to the receive.Listener interface.
type fileTransferListener struct {
	*manager
}

// Hear is called when a new file transfer is received. It creates a new
// internal received file transfer and starts waiting to receive file part
// messages.
func (ftl *fileTransferListener) Hear(msg receive.Message) {
	// Unmarshal the request message
	newFT := &NewFileTransfer{}
	err := proto.Unmarshal(msg.Payload, newFT)
	if err != nil {
		jww.ERROR.Printf(errProtoUnmarshal, err)
		return
	}
	// Generate new transfer ID
	rng := ftl.rng.GetStream()
	tid, err := ftCrypto.NewTransferID(rng)
	if err != nil {
		jww.ERROR.Printf(errNewRtTransferID, err)
		return
	}
	rng.Close()

	key := ftCrypto.UnmarshalTransferKey(newFT.TransferKey)
	numParts := uint16(newFT.GetNumParts())
	numFps := calcNumberOfFingerprints(int(numParts), newFT.GetRetry())

	rt, err := ftl.received.AddTransfer(&key, &tid, newFT.GetFileName(),
		newFT.GetTransferMac(), numParts, numFps, newFT.GetSize())
	if err != nil {
		jww.ERROR.Printf(errAddNewRt, tid, newFT.GetFileName(), err)
	}

	ftl.addFingerprints(rt)

	// Call the reception callback
	go ftl.receiveCB(&tid, newFT.GetFileName(), newFT.GetFileType(), msg.Sender,
		newFT.GetSize(), newFT.GetPreview())
}

// addFingerprints adds all fingerprints for unreceived parts in the received
// transfer.
func (m *manager) addFingerprints(rt *store.ReceivedTransfer) {
	// Build processor for each file part and add its fingerprint to receive on
	for _, c := range rt.GetUnusedCyphers() {
		p := &processor{
			Cypher:           c,
			ReceivedTransfer: rt,
			manager:          m,
		}

		err := m.cmix.AddFingerprint(m.myID, c.GetFingerprint(), p)
		if err != nil {
			jww.ERROR.Printf("[FT] Failed to add fingerprint for transfer "+
				"%s: %+v", rt.TransferID(), err)
		}
	}
}

// Name returns a name used for debugging.
func (ftl *fileTransferListener) Name() string {
	return listenerName
}
