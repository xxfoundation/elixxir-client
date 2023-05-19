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
	"fmt"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

// headerVersion is the most up-to-date edition of the header object. If the
// header object goes through significant restructuring, the version should be
// incremented. It is up to the developer to support older versions.
const headerVersion = 0

// Error messages.
const (
	entryDoesNotExistErr      = "entry for key %s could not be found."
	headerUnexpectedSerialErr = "unexpected data in serialized header"
	headerDecodeErr           = "failed to decode header info: %+v"
	headerUnmarshalErr        = "failed to unmarshal header json: %+v"
	headerMarshalErr          = "failed to marshal header: %+v"
	delimiter                 = "\n"
)

var delimiterBytes = []byte(delimiter)

// newHeader is the constructor of a header object.
func newHeader(DeviceID InstanceID) *header {
	return &header{
		Version:  headerVersion,
		DeviceID: DeviceID,
	}
}

// header is the object to which header strictly adheres. header serves as the
// marshal-able an unmarshal-able object that header.MarshalJSON and
// header.UnmarshalJSON utilizes when calling json.Marshal/json.Unmarshal.
type header struct {
	Version  uint16     `json:"version"`
	DeviceID InstanceID `json:"device"`
}

// serialize serializes a header object.
//
// Use deserializeHeader to reverse this operation.
func (h *header) serialize() ([]byte, error) {
	headerMarshal, err := json.Marshal(h)
	if err != nil {
		return nil, errors.Errorf(headerMarshalErr, err)
	}
	hdrBytes := make([]byte, len(xxdkTxLogHeader)+len(headerMarshal))
	copy(hdrBytes, []byte(xxdkTxLogHeader))
	copy(hdrBytes[len(xxdkTxLogHeader):], headerMarshal)
	return hdrBytes, nil
}

// deserializeHeader will deserialize header byte data.
//
// This is the inverse operation of header.serialize.
func deserializeHeader(headerSerial []byte) (*header, error) {

	if !bytes.HasPrefix(headerSerial, []byte(xxdkTxLogHeader)) {
		return nil, fmt.Errorf(headerUnexpectedSerialErr+" %v != %v",
			xxdkTxLogHeader, headerSerial[:len(xxdkTxLogHeader)])
	}

	headerMarshalled := headerSerial[len(xxdkTxLogHeader):]

	// Unmarshal header
	hdr := &header{}
	if err := json.Unmarshal(headerMarshalled, hdr); err != nil {
		return nil, errors.Errorf(headerUnmarshalErr, err)
	}

	return hdr, nil
}

func buildFile(h *header, ecrBody []byte) []byte {
	hSerial, err := h.serialize()
	if err != nil {
		jww.FATAL.Panicf("Failed to serialize the header")
	}
	bdy := make([]byte, base64.RawStdEncoding.EncodedLen(len(ecrBody)))
	base64.RawStdEncoding.Encode(bdy, ecrBody)

	buf := make([]byte, len(hSerial)+len(bdy)+1)
	copy(buf, hSerial)
	copy(buf[len(hSerial):], delimiterBytes)
	copy(buf[len(hSerial)+1:], bdy)
	return buf
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
	bdy := make([]byte, base64.RawStdEncoding.DecodedLen(len(ecrBody)))
	_, err = base64.RawStdEncoding.Decode(bdy, ecrBody)
	if err != nil {
		return nil, nil, err
	}
	return h, bdy, nil
}
