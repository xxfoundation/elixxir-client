package parse

import (
	"bytes"
	"gitlab.com/elixxir/client/interfaces/message"
	"testing"
	"time"
)

func TestnewFirstMessagePart(t *testing.T) {
	now := time.Now()
	fmp := newFirstMessagePart(message.Text, uint32(6), uint8(2), now,
		[]byte{'t', 'e', 's', 't', 'i', 'n', 'g'})

	if fmp.GetType() != message.Text {

	}

	if fmp.GetNumParts() != uint8(2) {

	}

	recorded_now, err := fmp.GetTimestamp()
	if err != nil {
		t.Fatal(err)
	}
	if recorded_now != now {

	}

	if !bytes.Equal(fmp.Bytes(), []byte{'t', 'e', 's', 't', 'i', 'n', 'g'}) {

	}
}
