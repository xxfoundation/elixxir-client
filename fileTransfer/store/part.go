////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"gitlab.com/elixxir/client/fileTransfer/store/cypher"
	"gitlab.com/elixxir/client/fileTransfer/store/fileMessage"
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

// MarkArrived marks the part as arrived. This should be called after the round
// the part is sent on succeeds.
func (p *Part) MarkArrived() {
	p.transfer.markArrived(p.partNum)
}

// Recipient returns the recipient of the file transfer.
func (p *Part) Recipient() *id.ID {
	return p.transfer.recipient
}

// TransferID returns the ID of the file transfer.
func (p *Part) TransferID() *ftCrypto.TransferID {
	return p.transfer.tid
}

// FileName returns the name of the file.
func (p *Part) FileName() string {
	return p.transfer.FileName()
}
