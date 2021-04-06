///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package message

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

// WARNING: Unsafe
// Payloads are not End to End encrypted, MetaData is NOT protected with
// this call, see SendE2E for End to End encryption and full privacy protection
// Internal SendUnsafe which bypasses the network check, will attempt to send to
// the network without checking state.
// This partitions payloads into multi-part messages but does NOT end to encrypt
// them
// Sends using SendCMIX and returns a list of rounds the messages are in. Will
// return an error if a single part of the message fails to send.
func (m *Manager) SendUnsafe(msg message.Send, param params.Unsafe) ([]id.Round, error) {
	if msg.MessageType == message.Raw {
		return nil, errors.Errorf("Raw (%d) is a reserved message type",
			msg.MessageType)
	}
	//timestamp the message
	ts := netTime.Now()

	//partition the message
	partitions, _, err := m.partitioner.Partition(msg.Recipient, msg.MessageType, ts,
		msg.Payload)

	if err != nil {
		return nil, errors.WithMessage(err, "failed to send unsafe message")
	}

	//send the partitions over cmix
	roundIds := make([]id.Round, len(partitions))
	errCh := make(chan error, len(partitions))

	wg := sync.WaitGroup{}

	jww.INFO.Printf("Unsafe sending %d messages to %s",
		len(partitions), msg.Recipient)

	for i, p := range partitions {
		myID := m.Session.User().GetCryptographicIdentity()
		msgCmix := format.NewMessage(m.Session.Cmix().GetGroup().GetP().ByteLen())
		msgCmix.SetContents(p)
		e2e.SetUnencrypted(msgCmix, myID.GetReceptionID())

		jww.INFO.Printf("Unsafe sending %d/%d to %s with msgDigest: %s",
			i+i, len(partitions), msg.Recipient, msgCmix.Digest())

		wg.Add(1)
		go func(i int) {
			var err error
			roundIds[i], _, err = m.SendCMIX(msgCmix, msg.Recipient, param.CMIX)
			if err != nil {
				errCh <- err
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	//see if any parts failed to send
	numFail, errRtn := getSendErrors(errCh)
	if numFail > 0 {
		jww.INFO.Printf("Failed to Unsafe send %d/%d to %s",
			numFail, len(partitions), msg.Recipient)
		return nil, errors.Errorf("Failed to send %v/%v sub payloads:"+
			" %s", numFail, len(partitions), errRtn)
	} else {
		jww.INFO.Printf("Sucesfully Unsafe sent %d/%d to %s",
			len(partitions)-numFail, len(partitions), msg.Recipient)
	}

	//return the rounds if everything send successfully
	return roundIds, nil
}

//returns any errors on the error channel
func getSendErrors(c chan error) (int, string) {
	var errRtn string
	numFail := 0
	done := false
	for !done {
		select {
		case err := <-c:
			errRtn += err.Error()
			numFail++
		default:
			done = true
		}
	}
	return numFail, errRtn
}
