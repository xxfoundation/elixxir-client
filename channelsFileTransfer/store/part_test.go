////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"bytes"
	"math/rand"
	"testing"

	"gitlab.com/elixxir/client/v4/channelsFileTransfer/store/fileMessage"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
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

// Tests that Part.PartNum returns the correct part index.
func TestPart_PartNum(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(25, t)
	partNum := uint16(5)
	part := st.GetUnsentParts()[partNum]

	part.MarkSent()

	if part.PartNum() != partNum {
		t.Errorf("Part #%d does not have expected partnum: %d.",
			partNum, part.PartNum())
	}
}

// Tests that Part.MarkSent correctly marks the part's status in the
// SentTransfer's partStatus vector.
func TestPart_MarkSent(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(25, t)
	partNum := uint16(0)
	part := st.GetUnsentParts()[partNum]

	part.MarkSent()

	if st.partStatus.Get(partNum) != uint8(SentPart) {
		t.Errorf("Part #%d not marked as sent.", partNum)
	}
}

// Tests that Part.MarkReceived correctly marks the part's status in the
// SentTransfer's partStatus vector.
func TestPart_MarkReceived(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(25, t)
	partNum := uint16(0)
	part := st.GetUnsentParts()[partNum]

	part.MarkSent()
	part.MarkReceived()

	if st.partStatus.Get(partNum) != uint8(ReceivedPart) {
		t.Errorf("Part #%d not marked as received.", partNum)
	}
}

// Tests that Part.MarkReceived returns the correct status for a part as its
// status is changed.
func TestPart_GetStatus(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(25, t)
	part := st.GetUnsentParts()[0]

	status := part.GetStatus()
	if status != UnsentPart {
		t.Errorf("Did not get expected status for part %d."+
			"\nexpected: %s\nreceived: %s", part.PartNum(), UnsentPart, status)
	}

	part.MarkSent()
	status = part.GetStatus()
	if status != SentPart {
		t.Errorf("Did not get expected status for part %d."+
			"\nexpected: %s\nreceived: %s", part.PartNum(), SentPart, status)
	}

	part.MarkReceived()
	status = part.GetStatus()
	if status != ReceivedPart {
		t.Errorf("Did not get expected status for part %d."+
			"\nexpected: %s\nreceived: %s", part.PartNum(), ReceivedPart, status)
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

// Tests that Part.FileID returns the correct file ID.
func TestPart_FileID(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(25, t)
	part := st.GetUnsentParts()[0]

	if part.FileID() != st.FileID() {
		t.Errorf("File ID does not match expected."+
			"\nexpected: %s\nreceived: %s", st.FileID(), part.FileID())
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

// Consistency test of Part.String.
func TestPart_String_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	fileData := make([]byte, 64)

	expectedStrings := []string{
		"{GmeTCfxGOqRqeIDPGDFroTglaY5zUwwxc9aRbeIf3Co= 929}",
		"{cwcVZ9hC9FUMwED+deZbcji/1kU4P9snnWeSZrxpdII= 82}",
		"{88nPaAIaeHf6yGtI6s+bjZFgk1mtG6g1iNlrWXmxhEQ= 982}",
		"{Fx8bvKcPXgZzXARA4fzl0BShwWVB5vXtJE6eQ+ud/Kk= 235}",
		"{gI/K4PcPbAr1KagBzQniMwE0T3XQ9RXxnVTWXYp0KEw= 978}",
		"{v4OhpMMKJblekaHCkDRh3P3MdTlogp6lN2+uLeUNe00= 504}",
		"{4KtkZgM0PbLWL041uRPzG13VomfFehiIC+7Je9CnM80= 566}",
		"{Ekflpke4eEK0vgZhSWXvPNEw8DkFvlgC3W1/3mXZRbY= 323}",
		"{ZIM//A0u3tEzb+1a/t+APedVMhSOX89ddAXZhaBiNyk= 198}",
		"{32Bpz2rsKJeSupgNkM88jLztIRLKYlMDyHx6sez57LU= 504}",
	}

	for i, expected := range expectedStrings {
		prng.Read(fileData)
		fid := ftCrypto.NewID(fileData)
		p := &Part{&SentTransfer{fid: fid}, nil, uint16(prng.Intn(1000))}

		if expected != p.String() {
			t.Errorf("Unexpected Part string (%d).\nexpected: %s\nreceived: %s",
				i, expected, p)
		}
	}
}
