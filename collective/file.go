////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"strings"
)

// headerVersion is the most up-to-date edition of the header object. If the
// header object goes through significant restructuring, the version should be
// incremented. It is up to the developer to support older versions.
const headerVersion = 0

// Error messages.
const (
	entryDoesNotExistErr      = "entry for key %s could not be found."
	headerUnexpectedSerialErr = "unexpected data in serialized header."
	headerDecodeErr           = "failed to decode header info: %+v"
	headerUnmarshalErr        = "failed to unmarshal header json: %+v"
	headerMarshalErr          = "failed to marshal header: %+v"
	delimiter                 = "\n"
)

var delimiterBytes = []byte(delimiter)

// newHeader is the constructor of a header object.
func newHeader(DeviceID cmix.InstanceID) *header {
	return &header{
		Version:  headerVersion,
		DeviceID: DeviceID,
	}
}

// header is the object to which header strictly adheres. header serves as the
// marshal-able an unmarshal-able object that header.MarshalJSON and
// header.UnmarshalJSON utilizes when calling json.Marshal/json.Unmarshal.
type header struct {
	Version  uint16          `json:"version"`
	DeviceID cmix.InstanceID `json:"device"`
}

// serialize serializes a header object.
//
// Use deserializeHeader to reverse this operation.
func (h *header) serialize() ([]byte, error) {

	// Marshal header into JSON
	headerMarshal, err := json.Marshal(h)
	if err != nil {
		return nil, errors.Errorf(headerMarshalErr, err)
	}

	// Construct header info
	headerInfo := xxdkTxLogHeader + base64.URLEncoding.EncodeToString(headerMarshal)

	return []byte(headerInfo), nil
}

// deserializeHeader will deserialize header byte data.
//
// This is the inverse operation of header.serialize.
func deserializeHeader(headerSerial []byte) (*header, error) {
	// Extract the header
	splitter := strings.Split(string(headerSerial), xxdkTxLogHeader)
	if len(splitter) != 2 {
		return nil, errors.Errorf(headerUnexpectedSerialErr)
	}

	// Decode header
	headerInfo, err := base64.URLEncoding.DecodeString(splitter[1])
	if err != nil {
		return nil, errors.Errorf(headerDecodeErr, err)
	}

	// Unmarshal header
	hdr := &header{}
	if err = json.Unmarshal(headerInfo, hdr); err != nil {
		return nil, errors.Errorf(headerUnmarshalErr, err)
	}

	return hdr, nil
}

func buildFile(h *header, ecrBody []byte) []byte {
	hSerial, err := h.serialize()
	if err != nil {
		jww.FATAL.Panicf("Failed to serialize the header")
	}

	ecrBody = []byte(base64.URLEncoding.EncodeToString(ecrBody))

	file := make([]byte, len(hSerial)+len(ecrBody)+len(delimiterBytes))
	//header first
	copy(file[:len(hSerial)], hSerial)
	//newline after
	copy(file[len(hSerial):len(hSerial)+len(delimiterBytes)], delimiterBytes)
	//body after
	copy(file[len(hSerial)+len(delimiterBytes):], ecrBody)
	return file
}

func decodeFile(file []byte) (*header, []byte, error) {
	read := bufio.NewReader(bytes.NewReader(file))
	headerBytes, _, err := read.ReadLine()
	if err != nil {
		return nil, nil, err
	}
	h, err := deserializeHeader(headerBytes)
	if err != nil {
		return nil, nil, err
	}
	ecrBody, _, err := read.ReadLine()
	if err != nil {
		return nil, nil, err
	}
	return h, ecrBody, nil
}
