////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcastFileTransfer

import (
	"fmt"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/broadcastFileTransfer/store"
	"gitlab.com/elixxir/client/broadcastFileTransfer/store/cypher"
	"gitlab.com/elixxir/client/broadcastFileTransfer/store/fileMessage"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/primitives/format"
)

// Error messages.
const (
	// processor.Process
	errDecryptPart   = "[FT] Failed to decrypt file part for transfer %s (%q) on round %d: %+v"
	errUnmarshalPart = "[FT] Failed to unmarshal decrypted file part for transfer %s (%q) on round %d: %+v"
	errAddPart       = "[FT] Failed to add part #%d to transfer transfer %s (%q): %+v"
)

// processor manages the reception of file transfer messages. Adheres to the
// message.Processor interface.
type processor struct {
	cypher.Cypher
	*store.ReceivedTransfer
	*manager
}

// Process decrypts and hands off the file part message and adds it to the
// correct file transfer.
func (p *processor) Process(msg format.Message,
	_ receptionID.EphemeralIdentity, round rounds.Round) {

	decryptedPart, err := p.Decrypt(msg)
	if err != nil {
		jww.ERROR.Printf(
			errDecryptPart, p.TransferID(), p.FileName(), round.ID, err)
		return
	}

	partMsg, err := fileMessage.UnmarshalPartMessage(decryptedPart)
	if err != nil {
		jww.ERROR.Printf(
			errUnmarshalPart, p.TransferID(), p.FileName(), round.ID, err)
		return
	}

	err = p.AddPart(partMsg.GetPart(), int(partMsg.GetPartNum()))
	if err != nil {
		jww.WARN.Printf(
			errAddPart, partMsg.GetPartNum(), p.TransferID(), p.FileName(), err)
		return
	}

	if p.NumParts() == p.NumReceived() {
		jww.DEBUG.Printf("[FT] Completed received transfer %s.", p.TransferID())
	}

	jww.TRACE.Printf("[FT] Received part %d of %d of transfer %s on round %d",
		partMsg.GetPartNum(), p.NumParts()-1, p.TransferID(), round.ID)

	// Call callback with updates
	p.callbacks.Call(p.TransferID(), nil)
}

func (p *processor) String() string {
	return fmt.Sprintf("FileTransfer(%s)", p.Recipient())
}
