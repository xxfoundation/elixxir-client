////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package parse

import (
	"encoding/binary"
	"errors"
	"gitlab.com/privategrity/client/cmixproto"
)

// To allow mobile to bind this module if necessary, we'll return the two parts
// of a body in a struct
type TypedBody struct {
	Type cmixproto.Type
	Body []byte
}

// Determine the type of a message body. Returns the type and the part of the
// body that doesn't include the type.
func Parse(body []byte) (*TypedBody, error) {
	messageType, numBytesRead := binary.Uvarint(body)
	if numBytesRead < 0 {
		return nil, errors.New("Body type parse: Type too long. " +
			"Set a byte's most significant bit to 0 within the first 8 bytes.")
	}
	result := &TypedBody{}
	result.Type = cmixproto.Type(messageType)
	result.Body = body[numBytesRead:]
	return result, nil
}

// Pack this message for the network
func Pack(body *TypedBody) []byte {
	// Assumes that the underlying type of cmixproto.Type is int32
	return append(TypeAsBytes(int32(body.Type)), body.Body...)
}

// Mobile or other packages can use this wrapper to easily determine the
// correct magic number for a type
func TypeAsBytes(messageType int32) []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, uint64(messageType))
	return buf[:n]
}
