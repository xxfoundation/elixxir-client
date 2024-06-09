////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found         //
// in the LICENSE file                                                        //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMsgPartitions(t *testing.T) {
	headerSize := (900 - handshakeCiphertextOverhead)
	otherSize := (900 - channelCiphertextOverhead)

	// A very long message, lets say 13 otherSizes
	// (which should make 14 parts as header is smallerx)
	msg := make([]byte, otherSize*13)
	patternBuf := []byte{'a', 'b', 'c', 'd', 'e', 'f', 'g'}
	for i := 0; i < len(msg); i++ {
		msg[i] = patternBuf[i%len(patternBuf)]
	}
	parts := partitionMessage(msg, headerSize, otherSize)
	require.Equal(t, len(parts), 14)
	msgOut, err := reconstructPartitions(parts)
	require.NoError(t, err)
	require.Equal(t, msg, msgOut)

	// A short message
	msg = []byte("Hello")
	parts = partitionMessage(msg, headerSize, otherSize)
	require.Equal(t, len(parts), 1)
	msgOut, err = reconstructPartitions(parts)
	require.NoError(t, err)
	require.Equal(t, msg, msgOut)

	// A message 1 character longer than header
	msg = make([]byte, headerSize+1)
	for i := 0; i < len(msg); i++ {
		msg[i] = patternBuf[i%len(patternBuf)]
	}
	parts = partitionMessage(msg, headerSize, otherSize)
	require.Equal(t, len(parts), 2)
	msgOut, err = reconstructPartitions(parts)
	require.NoError(t, err)
	require.Equal(t, msg, msgOut)

	// A message exactly the header size-1, which will still be 2 long
	msg = make([]byte, headerSize-1)
	for i := 0; i < len(msg); i++ {
		msg[i] = patternBuf[i%len(patternBuf)]
	}
	parts = partitionMessage(msg, headerSize, otherSize)
	require.Equal(t, len(parts), 2)
	msgOut, err = reconstructPartitions(parts)
	require.NoError(t, err)
	require.Equal(t, msg, msgOut)

	// A message exactly the header size-2 which will fit
	msg = make([]byte, headerSize-2)
	for i := 0; i < len(msg); i++ {
		msg[i] = patternBuf[i%len(patternBuf)]
	}
	parts = partitionMessage(msg, headerSize, otherSize)
	require.Equal(t, len(parts), 1)
	msgOut, err = reconstructPartitions(parts)
	require.NoError(t, err)
	require.Equal(t, msg, msgOut)

	// A message exactly header+other size - 2
	msg = make([]byte, headerSize+otherSize-2)
	for i := 0; i < len(msg); i++ {
		msg[i] = patternBuf[i%len(patternBuf)]
	}
	parts = partitionMessage(msg, headerSize, otherSize)
	require.Equal(t, len(parts), 2)
	msgOut, err = reconstructPartitions(parts)
	require.NoError(t, err)
	require.Equal(t, msg, msgOut)
}
