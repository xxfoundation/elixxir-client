////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/fileTransfer/store"
	"gitlab.com/elixxir/client/v5/stoppable"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"go.uber.org/ratelimit"
	"time"
)

const (
	// Duration to wait before adding a partially filled part packet to the send
	// channel.
	unfilledPacketTimeout = 100 * time.Millisecond
)

// batchBuilderThread creates batches of file parts as they become available and
// buffer them to send. Also rate limits adding to the buffer.
func (m *manager) batchBuilderThread(stop *stoppable.Single) {
	jww.INFO.Printf("[FT] Starting batch builder thread.")
	// Calculate rate and make rate limiter
	rl := newRateLimiter(m.params.MaxThroughput, m.cmixGroup)

	// Build each batch and add to the queue
	for {
		numParts := generateRandomPacketSize(m.rng)
		packet := make([]store.Part, 0, numParts)
		delayedTimer := NewDelayedTimer(unfilledPacketTimeout)
	loop:
		for cap(packet) > len(packet) {
			select {
			case <-stop.Quit():
				delayedTimer.Stop()
				jww.DEBUG.Printf("[FT] Stopping file part packing thread " +
					"while packing: stoppable triggered.")
				stop.ToStopped()
				return
			case <-*delayedTimer.C:
				break loop
			case p := <-m.batchQueue:
				packet = append(packet, p)
				delayedTimer.Start()
			}
		}

		// Rate limiter
		rl.Take()
		m.sendQueue <- packet
	}
}

// newRateLimiter generates a new ratelimit.Limiter that limits the bandwidth to
// the given max throughput (in bytes per second).
func newRateLimiter(
	maxThroughput int, cmixGroup *cyclic.Group) ratelimit.Limiter {
	// Calculate rate and make rate limiter if max throughput is set
	if maxThroughput > 0 {
		// Calculate the average amount of data sent in each batch
		messageSize := format.NewMessage(cmixGroup.GetP().ByteLen()).ContentsSize()
		avgNumMessages := (minPartsSendPerRound + maxPartsSendPerRound) / 2
		avgSendSize := avgNumMessages * messageSize

		jww.DEBUG.Printf("[FT] Rate limiting parameters: message size: %d, "+
			"average number of messages per send: %d, average size of send: %d",
			messageSize, avgNumMessages, avgSendSize)

		// Calculate the time window needed to achieve the desired bandwidth
		per := time.Second
		switch {
		case avgSendSize < maxThroughput:
			per = time.Second
		case avgSendSize < maxThroughput*60:
			per = time.Minute
		case avgSendSize < maxThroughput*60*60:
			per = time.Hour
		case avgSendSize < maxThroughput*60*60*24:
			per = time.Hour * 24
		case avgSendSize < maxThroughput*60*60*24*7:
			per = time.Hour * 24 * 7
		}

		// Calculate the rate of messages per time window
		rate := int((float64(maxThroughput) / float64(avgSendSize)) *
			float64(per/time.Second))

		jww.INFO.Printf("[FT] Max throughput is %d bytes/second. "+
			"File transfer will be rate limited to %d per %s.",
			maxThroughput, rate, per)

		return ratelimit.New(rate, ratelimit.WithoutSlack, ratelimit.Per(per))
	}

	// If the max throughput is zero, then create an unlimited rate limiter
	jww.WARN.Printf("[FT] Max throughput is %d bytes/second. "+
		"File transfer will not be rate limited.", maxThroughput)
	return ratelimit.NewUnlimited()
}

// generateRandomPacketSize returns a random number between minPartsSendPerRound
// and maxPartsSendPerRound, inclusive.
func generateRandomPacketSize(rngGen *fastRNG.StreamGenerator) int {
	rng := rngGen.GetStream()
	defer rng.Close()

	// Generate random bytes
	b, err := csprng.Generate(8, rng)
	if err != nil {
		jww.FATAL.Panicf(getRandomNumPartsRandPanic, err)
	}

	// Convert bytes to integer
	num := binary.LittleEndian.Uint64(b)

	// Return random number that is minPartsSendPerRound <= num <= max
	return int((num % (maxPartsSendPerRound)) + minPartsSendPerRound)
}
