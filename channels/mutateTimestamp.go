////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/crypto/large"
	"time"
)

const (
	// use a prime close to one million to ensure patterns dont arise due to
	// cofactors with the message ID when doing the modulo
	tenMsInNs     = 10000019
	halfTenMsInNs = tenMsInNs / 2
)

var tenMsInNsLargeInt = large.NewInt(tenMsInNs)

// mutateTimestamp is used to modify the the timestamps on all messages in a
// deterministic manner. This is because message ordering is done by timestamp
// and the timestamps come from the rounds, which means multiple messages can
// have the same timestamp due to being in the same round. The meaning of
// conversations can change depending on order, so while no explicit order
// can be discovered because to do so can leak potential ordering info for the
// mix, choosing an arbitrary order and having all clients agree will at least
// ensure that misunderstandings due to disagreements in order cannot occur
//
// In order to do this, this function mutates the timestamp of the round within
// +/- 5ms seeded based upon the message ID.
// It should be noted that this is only a reasonable assumption when the number
// of messages in a channel isn't too much. For example, under these conditions
// the birthday paradox of getting a collision if there are 10 messages for the
// channel in the same round is ~4*10^-6, but the chance if there are 50
// messages is 10^-4, and if the entire round is full of messages for the
// channel (1000 messages), .0487.
func mutateTimestamp(ts time.Time, msgID channel.MessageID) time.Time {

	// Treat the message ID as a number and mod it by the number of ns in an ms
	// to get an offset factor. Use a prime close to 1000000 to make sure there
	// are no patterns in the output and reduce the chance of collision. While
	// the fields do not align, so there is some bias towards some parts of the
	// output field, that bias is too small to matter because log2(10000019) ~23
	// while the input field is 256.
	offsetLarge := large.NewIntFromBytes(msgID.Bytes())
	offsetLarge.Mod(offsetLarge, tenMsInNsLargeInt)

	// subtract half the field size so on average (across many runs) the message
	// timestamps are not changed
	offset := offsetLarge.Int64() - halfTenMsInNs

	return time.Unix(0, ts.UnixNano()+offset)
}
