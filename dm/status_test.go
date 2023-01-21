////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStatusString strings should never change, so lock them with a test.
func TestStatusString(t *testing.T) {
	unsent := Unsent
	sent := Sent
	received := Received
	failed := Failed

	invalid := Status(4)

	require.Equal(t, unsent.String(), "unsent")
	require.Equal(t, sent.String(), "sent")
	require.Equal(t, received.String(), "received")
	require.Equal(t, failed.String(), "failed")
	require.Equal(t, invalid.String(), "Invalid SentStatus: 4")
}
