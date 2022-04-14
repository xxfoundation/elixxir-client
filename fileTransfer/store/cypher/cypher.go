////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cypher

import (
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
)

// Cypher contains the cryptographic and identifying information needed to
// decrypt a file part and associate it with the correct file.
type Cypher struct {
	*Manager
	fpNum uint16
}

// Encrypt encrypts the file part contents and returns them along with a MAC and
// fingerprint.
func (c Cypher) Encrypt(contents []byte) (
	cipherText, mac []byte, fp format.Fingerprint) {

	// Generate fingerprint
	fp = ftCrypto.GenerateFingerprint(*c.key, c.fpNum)

	// Encrypt part and get MAC
	cipherText, mac = ftCrypto.EncryptPart(*c.key, contents, c.fpNum, fp)

	return cipherText, mac, fp
}

// Decrypt decrypts the content of the message.
func (c Cypher) Decrypt(msg format.Message) ([]byte, error) {
	filePart, err := ftCrypto.DecryptPart(
		*c.key, msg.GetContents(), msg.GetMac(), c.fpNum, msg.GetKeyFP())
	if err != nil {
		return nil, err
	}

	c.fpVector.Use(uint32(c.fpNum))

	return filePart, nil
}

// GetFingerprint generates and returns the fingerprints.
func (c Cypher) GetFingerprint() format.Fingerprint {
	return ftCrypto.GenerateFingerprint(*c.key, c.fpNum)
}
