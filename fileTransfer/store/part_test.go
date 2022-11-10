////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"bytes"
	"gitlab.com/elixxir/client/v4/fileTransfer/store/fileMessage"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
	"testing"
)

// Tests that the encrypted part returned by Part.GetEncryptedPart can be
// decrypted, unmarshalled, and that it matches the original.
func TestPart_GetEncryptedPart(t *testing.T) {
	st, parts, key, _, _ := newTestSentTransfer(25, t)
	partNum := 0
	part := st.GetUnsentParts()[partNum]

	encryptedPart, mac, fp, err := part.GetEncryptedPart(
		format.NewMessage(numPrimeBytes).ContentsSize())
	if err != nil {
		t.Errorf("GetEncryptedPart returned an error: %+v", err)
	}

	decryptedPart, err := ftCrypto.DecryptPart(
		*key, encryptedPart, mac, uint16(partNum), fp)
	if err != nil {
		t.Errorf("Failed to decrypt part: %+v", err)
	}

	partMsg, err := fileMessage.UnmarshalPartMessage(decryptedPart)
	if err != nil {
		t.Errorf("Failed to unmarshal part message: %+v", err)
	}

	if !bytes.Equal(parts[partNum], partMsg.GetPart()) {
		t.Errorf("Decrypted part does not match original."+
			"\nexpected: %q\nreceived: %q", parts[partNum], partMsg.GetPart())
	}

	if int(partMsg.GetPartNum()) != partNum {
		t.Errorf("Decrypted part does not have correct part number."+
			"\nexpected: %d\nreceived: %d", partNum, partMsg.GetPartNum())
	}
}

// Tests that Part.GetEncryptedPart returns an error when the underlying cypher
// manager runs out of fingerprints.
func TestPart_GetEncryptedPart_OutOfFingerprints(t *testing.T) {
	numParts := uint16(25)
	st, _, _, numFps, _ := newTestSentTransfer(numParts, t)
	part := st.GetUnsentParts()[0]
	for i := uint16(0); i < numFps; i++ {
		_, _, _, err := part.GetEncryptedPart(
			format.NewMessage(numPrimeBytes).ContentsSize())
		if err != nil {
			t.Errorf("Getting encrtypted part %d failed: %+v", i, err)
		}
	}

	_, _, _, err := part.GetEncryptedPart(
		format.NewMessage(numPrimeBytes).ContentsSize())
	if err == nil {
		t.Errorf("Failed to get an error when run out of fingerprints.")
	}
}

// Tests that Part.MarkArrived correctly marks the part's status in the
// SentTransfer's partStatus vector.
func TestPart_MarkArrived(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(25, t)
	partNum := 0
	part := st.GetUnsentParts()[partNum]

	part.MarkArrived()

	if !st.partStatus.Used(uint32(partNum)) {
		t.Errorf("Part #%d not marked as arrived.", partNum)
	}
}

// Tests that Part.Recipient returns the correct recipient ID.
func TestPart_Recipient(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(25, t)
	part := st.GetUnsentParts()[0]

	if !part.Recipient().Cmp(st.Recipient()) {
		t.Errorf("Recipient ID does not match expected."+
			"\nexpected: %s\nreceived: %s", st.Recipient(), part.Recipient())
	}
}

// Tests that Part.TransferID returns the correct transfer ID.
func TestPart_TransferID(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(25, t)
	part := st.GetUnsentParts()[0]

	if part.TransferID() != st.TransferID() {
		t.Errorf("Transfer ID does not match expected."+
			"\nexpected: %s\nreceived: %s", st.TransferID(), part.TransferID())
	}
}

// Tests that Part.FileName returns the correct file name.
func TestPart_FileName(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(25, t)
	part := st.GetUnsentParts()[0]

	if part.FileName() != st.FileName() {
		t.Errorf("File name does not match expected."+
			"\nexpected: %q\nreceived: %q", st.FileName(), part.FileName())
	}
}
