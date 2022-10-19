////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"io"
	"time"

	"github.com/pkg/errors"
)

var alreadyClosedErr = errors.New("connection is closed")

// Connection is a wrapper for the E2E and auth packages.
// It can be used to automatically establish an E2E partnership
// with a partner.Manager, or be built from an existing E2E partnership.
// You can then use this interface to send to and receive from the
// newly-established partner.Manager.
type Connection interface {
	// Closer deletes this Connection's partner.Manager and releases resources
	io.Closer

	// FirstPartitionSize returns the max partition payload size for the
	// first payload
	FirstPartitionSize() uint

	// SecondPartitionSize returns the max partition payload size for all
	// payloads after the first payload
	SecondPartitionSize() uint

	// PartitionSize returns the partition payload size for the given
	// payload index. The first payload is index 0.
	PartitionSize(payloadIndex uint) uint

	// PayloadSize Returns the max payload size for a partitionable E2E
	// message
	PayloadSize() uint

	// LastUse returns the timestamp of the last time the connection was
	// utilised.
	LastUse() time.Time
}

// Callback is the callback format required to retrieve
// new Connection objects as they are established.
type Callback func(connection Connection)
