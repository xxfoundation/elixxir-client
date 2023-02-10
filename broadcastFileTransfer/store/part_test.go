////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"bytes"
	"gitlab.com/elixxir/client/v4/broadcastFileTransfer/store/fileMessage"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"io"
	"math/rand"
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

// Consistency test of Part.String.
func TestPart_String_Consistency(t *testing.T) {
	randPrng := rand.New(rand.NewSource(42))
	prng := NewPrng(42)
	newTID := func() *ftCrypto.TransferID {
		tid, err := ftCrypto.NewTransferID(prng)
		if err != nil {
			t.Fatalf("Failed to created new transfer ID: %+v", err)
		}
		return &tid
	}

	tests := map[string]*Part{
		"{U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVI= 305}": {
			&SentTransfer{tid: newTID()}, nil, uint16(randPrng.Intn(1000))},
		"{39ebTXZCm2F6DJ+fDTulWwzA1hRMiIU1hBrL4HCbB1g= 987}": {
			&SentTransfer{tid: newTID()}, nil, uint16(randPrng.Intn(1000))},
		"{CD9h03W8ArQd9PkZKeGP2p5vguVOdI6B555LvW/jTNw= 668}": {
			&SentTransfer{tid: newTID()}, nil, uint16(randPrng.Intn(1000))},
		"{uoQ+6NY+jE/+HOvqVG2PrBPdGqwEzi6ih3xVec+ix44= 750}": {
			&SentTransfer{tid: newTID()}, nil, uint16(randPrng.Intn(1000))},
		"{GwuvrogbgqdREIpC7TyQPKpDRlp4YgYWl4rtDOPGxPM= 423}": {
			&SentTransfer{tid: newTID()}, nil, uint16(randPrng.Intn(1000))},
		"{rnvD4ElbVxL+/b4MECiH4QDazS2IX2kstgfaAKEcHHA= 345}": {
			&SentTransfer{tid: newTID()}, nil, uint16(randPrng.Intn(1000))},
		"{ceeWotwtwlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGE= 357}": {
			&SentTransfer{tid: newTID()}, nil, uint16(randPrng.Intn(1000))},
		"{SYlH/fNEQQ7UwRYCP6jjV2tv7Sf/iXS6wMr9mtBWkrE= 176}": {
			&SentTransfer{tid: newTID()}, nil, uint16(randPrng.Intn(1000))},
		"{NhnnOJZN/ceejVNDc2Yc/WbXT+weG4lJGrcjbkt1IWI= 128}": {
			&SentTransfer{tid: newTID()}, nil, uint16(randPrng.Intn(1000))},
		"{kM8r60LDyicyhWDxqsBnzqbov0bUqytGgEAsX7KCDog= 643}": {
			&SentTransfer{tid: newTID()}, nil, uint16(randPrng.Intn(1000))},
	}

	for expected, p := range tests {
		if expected != p.String() {
			t.Errorf("Unexpected Part string.\nexpected: %s\nreceived: %s",
				expected, p)
		}
	}
}

// Prng is a PRNG that satisfies the csprng.Source interface.
type Prng struct{ prng io.Reader }

func NewPrng(seed int64) csprng.Source     { return &Prng{rand.New(rand.NewSource(seed))} }
func (s *Prng) Read(b []byte) (int, error) { return s.prng.Read(b) }
func (s *Prng) SetSeed([]byte) error       { return nil }
