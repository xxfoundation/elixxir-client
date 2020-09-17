package network

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

// SendUnsafe sends an unencrypted payload to the provided recipient
// with the provided msgType. Returns the list of rounds in which parts
// of the message were sent or an error if it fails.
// NOTE: Do not use this function unless you know what you are doing.
// This function always produces an error message in client logging.
func (m *Manager) SendUnsafe(msg message.Send, param params.Unsafe) ([]id.Round, error) {
	if !m.health.IsRunning() {
		return nil, errors.New("cannot send unsafe message when the " +
			"network is not healthy")
	}

	jww.WARN.Println("Sending unsafe message. Unsafe payloads have no end" +
		" to end encryption, they have limited security and privacy " +
		"preserving properties")

	return m.sendUnsafe(msg, param)
}

func (m *Manager) sendUnsafe(msg message.Send, param params.Unsafe) ([]id.Round, error) {

	//timestamp the message
	ts := time.Now()

	//partition the message
	partitions, err := m.partitioner.Partition(msg.Recipient, msg.MessageType, ts,
		msg.Payload)

	if err != nil {
		return nil, errors.WithMessage(err, "failed to send unsafe message")
	}

	//send the partitions over cmix
	roundIds := make([]id.Round, len(partitions))
	errCh := make(chan error, len(partitions))

	wg := sync.WaitGroup{}

	for i, p := range partitions {
		msgCmix := format.NewMessage(m.Context.Session.Cmix().GetGroup().GetP().ByteLen())
		msgCmix.SetContents(p)
		e2e.SetUnencrypted(msgCmix, msg.Recipient)
		wg.Add(1)
		go func(i int) {
			var err error
			roundIds[i], err = m.sendCMIX(msgCmix, param.CMIX)
			if err != nil {
				errCh <- err
			}
			wg.Done()
		}(i)
	}

	//see if any parts failed to send
	numFail, errRtn := getSendErrors(errCh)
	if numFail > 0 {
		return nil, errors.Errorf("Failed to send %v/%v sub payloads:"+
			" %s", numFail, len(partitions), errRtn)
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
