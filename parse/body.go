package parse

import (
	"encoding/binary"
	"errors"
)

// To allow mobile to bind this module if necessary, we'll return the two parts
// of a body in a struct
type TypedBody struct {
	BodyType int64
	Body     []byte
}

// Determine the type of a message body. Returns the type and the part of the
// body that doesn't include the type.
// TODO test error cases where there are too many bytes with MSB set
func Parse(body []byte) (*TypedBody, error) {
	messageType, numBytesRead := binary.Uvarint(body)
	if numBytesRead < 0 {
		return nil, errors.New("Body type parse: Type too long. " +
			"Set a byte's most significant bit to 0 within the first 8 bytes.")
	}
	result := &TypedBody{}
	result.BodyType = int64(messageType)
	result.Body = body[numBytesRead:]
	return result, nil
}
