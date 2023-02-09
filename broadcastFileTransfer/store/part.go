////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"strconv"

	"gitlab.com/elixxir/client/v4/broadcastFileTransfer/store/cypher"
	"gitlab.com/elixxir/client/v4/broadcastFileTransfer/store/fileMessage"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// Part contains information about a single file part and its parent transfer.
// Also contains cryptographic information needed to encrypt the part data.
type Part struct {
	transfer      *SentTransfer
	cypherManager *cypher.Manager
	partNum       uint16
}

// GetEncryptedPart gets the specified part, encrypts it, and returns the
// encrypted part along with its MAC and fingerprint. An error is returned if no
// fingerprints are available.
func (p *Part) GetEncryptedPart(contentsSize int) (
	encryptedPart, mac []byte, fp format.Fingerprint, err error) {
	// Create new empty file part message of the size provided
	partMsg := fileMessage.NewPartMessage(contentsSize)

	// Add part number and part data to part message
	partMsg.SetPartNum(p.partNum)
	partMsg.SetPart(p.transfer.getPartData(p.partNum))

	// Get next cypher
	c, err := p.cypherManager.PopCypher()
	if err != nil {
		p.transfer.markTransferFailed()
		return nil, nil, format.Fingerprint{}, err
	}

	// Encrypt part and get MAC and fingerprint
	encryptedPart, mac, fp = c.Encrypt(partMsg.Marshal())

	return encryptedPart, mac, fp, nil
}

// PartNum returns the index of this part.
func (p *Part) PartNum() uint16 {
	return p.partNum
}

// MarkSent marks the part as sent. This should be called after the round the
// part is sent on succeeds.
func (p *Part) MarkSent() {
	p.transfer.markSent(p.partNum)
}

// MarkReceived marks the part as received. This should be called after the part
// has been received.
func (p *Part) MarkReceived() {
	p.transfer.markReceived(p.partNum)
}

// GetStatus returns the SentPartStatus of this part.
func (p *Part) GetStatus() SentPartStatus {
	return p.transfer.getPartStatus(p.partNum)
}

// Recipient returns the recipient of the file transfer.
func (p *Part) Recipient() *id.ID {
	return p.transfer.recipient
}

// FileID returns the ID of the file.
func (p *Part) FileID() ftCrypto.ID {
	return p.transfer.fid
}

// FileName returns the name of the file.
func (p *Part) FileName() string {
	return p.transfer.FileName()
}

// String returns a human-readable representation of a Part. Used for debugging.
func (p *Part) String() string {
	return "{" + p.transfer.fid.String() + " " + strconv.Itoa(int(p.partNum)) + "}"
}
