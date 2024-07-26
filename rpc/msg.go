////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found         //
// in the LICENSE file                                                        //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

var ErrMissingParts = errors.Errorf("need more parts")

// partitionMessage creates an ordered partition of messages based on the
// specified sizes. This function is intended to be used with plaintext, so
// users should take care to subtract the overheads from the cryptography
// for the corresponding header and other message sizes.
func partitionMessage(msg []byte, headerMsgSize, otherMsgSize uint64) [][]byte {
	// Header first
	headerMsg := make([]byte, headerMsgSize)

	// Prepend the whole message size, this allows the receiver to
	// reconstruct the message.
	var mSzBuf []byte
	mSzBuf = binary.AppendUvarint(mSzBuf, uint64(len(msg)))
	copy(headerMsg, mSzBuf)

	numSzBytes := uint64(len(mSzBuf))
	numMsgBytes := uint64(len(msg))
	firstPartSize := headerMsgSize - numSzBytes
	parts := make([][]byte, 1)

	// Msg fits into header
	if firstPartSize >= numMsgBytes {
		copy(headerMsg[numSzBytes:], msg)
		parts[0] = headerMsg
		return parts
	}

	// Msg does not fit into header
	copy(headerMsg[numSzBytes:], msg[:firstPartSize])
	parts[0] = headerMsg
	msgBytesLeft := numMsgBytes - firstPartSize

	// Add the rest of the parts
	numFullPartsLeft := msgBytesLeft / otherMsgSize
	offset := firstPartSize
	for i := uint64(0); i < numFullPartsLeft; i++ {
		parts = append(parts, msg[offset:offset+otherMsgSize])
		offset += otherMsgSize
	}

	// Last part (if necessary)
	if (msgBytesLeft % otherMsgSize) != 0 {
		p := make([]byte, otherMsgSize)
		copy(p, msg[offset:])
		parts = append(parts, p)
	}

	return parts
}

// reconstructPartitions recreates the original message sent to
// partitionMessage.
func reconstructPartitions(parts [][]byte) ([]byte, error) {
	if len(parts) == 0 {
		return nil, ErrMissingParts
	}

	headerReader := bytes.NewReader(parts[0])

	// read the size
	mSz, err := binary.ReadUvarint(headerReader)
	if err != nil {
		return nil, err
	}

	// Do we have the whole message yet?
	curSize := 0
	for i := 0; i < len(parts); i++ {
		curSize += len(parts[i])
	}
	if mSz > uint64(curSize) {
		jww.DEBUG.Printf("[RPC] still missing parts: %d > %d",
			mSz, curSize)
		return nil, ErrMissingParts
	}

	msg := make([]byte, curSize)
	offset, err := headerReader.Read(msg)
	if err != nil {
		return nil, err
	}

	for i := 1; i < len(parts); i++ {
		copy(msg[offset:], parts[i])
		offset += len(parts[i])
	}

	return msg[:mSz], nil
}
