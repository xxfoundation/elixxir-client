///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"encoding/json"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
)

const currentMeteredCmixMessageVersion = 0

type meteredCmixMessageHandler struct{}

type meteredCmixMessage struct {
	M         []byte
	Ri        []byte
	Identity  []byte
	Count     uint
	Timestamp time.Time
}

// SaveMessage saves the message as a versioned object at the specified key in
// the key value store.
func (*meteredCmixMessageHandler) SaveMessage(kv *versioned.KV, m interface{},
	key string) error {
	msg := m.(meteredCmixMessage)

	marshaled, err := json.Marshal(&msg)
	if err != nil {
		return errors.WithMessage(err, "Failed to marshal metered cmix message")
	}

	// Create versioned object
	obj := versioned.Object{
		Version:   currentMeteredCmixMessageVersion,
		Timestamp: netTime.Now(),
		Data:      marshaled,
	}

	// Save versioned object
	return kv.Set(key, utility.CurrentMessageBufferVersion, &obj)
}

// LoadMessage returns the message with the specified key from the key value
// store. An empty message and error are returned if the message could not be
// retrieved.
func (*meteredCmixMessageHandler) LoadMessage(kv *versioned.KV, key string) (
	interface{}, error) {
	// Load the versioned object
	vo, err := kv.Get(key, currentMeteredCmixMessageVersion)
	if err != nil {
		return nil, err
	}

	msg := meteredCmixMessage{}
	err = json.Unmarshal(vo.Data, &msg)
	if err != nil {
		return nil,
			errors.WithMessage(err, "Failed to unmarshal metered cmix message")
	}

	// Create message from data
	return msg, nil
}

// DeleteMessage deletes the message with the specified key from the key value
// store.
func (*meteredCmixMessageHandler) DeleteMessage(kv *versioned.KV, key string) error {
	return kv.Delete(key, currentMeteredCmixMessageVersion)
}

// HashMessage generates a hash of the message.
func (*meteredCmixMessageHandler) HashMessage(m interface{}) utility.MessageHash {
	h, _ := blake2b.New256(nil)

	h.Write(m.(meteredCmixMessage).M)
	h.Write(m.(meteredCmixMessage).Ri)
	h.Write(m.(meteredCmixMessage).Identity)

	var messageHash utility.MessageHash
	copy(messageHash[:], h.Sum(nil))

	return messageHash
}

// MeteredCmixMessageBuffer wraps the message buffer to store and load raw cMix
// messages.
type MeteredCmixMessageBuffer struct {
	mb  *utility.MessageBuffer
	kv  *versioned.KV
	key string
}

func NewMeteredCmixMessageBuffer(kv *versioned.KV, key string) (
	*MeteredCmixMessageBuffer, error) {
	mb, err := utility.NewMessageBuffer(kv, &meteredCmixMessageHandler{}, key)
	if err != nil {
		return nil, err
	}

	return &MeteredCmixMessageBuffer{mb: mb, kv: kv, key: key}, nil
}

func LoadMeteredCmixMessageBuffer(kv *versioned.KV, key string) (
	*MeteredCmixMessageBuffer, error) {
	mb, err := utility.LoadMessageBuffer(kv, &meteredCmixMessageHandler{}, key)
	if err != nil {
		return nil, err
	}

	return &MeteredCmixMessageBuffer{mb: mb, kv: kv, key: key}, nil
}

func NewOrLoadMeteredCmixMessageBuffer(kv *versioned.KV, key string) (
	*MeteredCmixMessageBuffer, error) {
	mb, err := utility.LoadMessageBuffer(kv, &meteredCmixMessageHandler{}, key)
	if err != nil {
		jww.WARN.Printf(
			"Failed to find MeteredCmixMessageBuffer %s, making a new one", key)
		return NewMeteredCmixMessageBuffer(kv, key)
	}

	return &MeteredCmixMessageBuffer{mb: mb, kv: kv, key: key}, nil
}

func (mcmb *MeteredCmixMessageBuffer) Add(m format.Message, ri *pb.RoundInfo,
	identity receptionID.EphemeralIdentity) (uint, time.Time) {
	if m.GetPrimeByteLen() == 0 {
		jww.FATAL.Panic(
			"Cannot handle a metered cMix message with a length of 0.")
	}
	jww.TRACE.Printf("Metered Messages Add(MsgDigest: %s)",
		m.Digest())

	msg := buildMsg(m, ri, identity)
	addedMsgFace := mcmb.mb.Add(msg)
	addedMessage := addedMsgFace.(meteredCmixMessage)

	return addedMessage.Count, addedMessage.Timestamp
}

func (mcmb *MeteredCmixMessageBuffer) AddProcessing(m format.Message,
	ri *pb.RoundInfo, identity receptionID.EphemeralIdentity) (uint, time.Time) {
	if m.GetPrimeByteLen() == 0 {
		jww.FATAL.Panic(
			"Cannot handle a metered cMix message with a length of 0.")
	}

	msg := buildMsg(m, ri, identity)
	addedMsgFace := mcmb.mb.AddProcessing(msg)
	addedMessage := addedMsgFace.(meteredCmixMessage)

	return addedMessage.Count, addedMessage.Timestamp
}

func (mcmb *MeteredCmixMessageBuffer) Next() (format.Message, *pb.RoundInfo,
	receptionID.EphemeralIdentity, bool) {
	m, ok := mcmb.mb.Next()
	if !ok {
		return format.Message{}, nil, receptionID.EphemeralIdentity{}, false
	}

	msg := m.(meteredCmixMessage)

	// Increment the count and save
	msg.Count++
	mcmh := &meteredCmixMessageHandler{}
	err := mcmh.SaveMessage(mcmb.kv, msg,
		utility.MakeStoredMessageKey(mcmb.key, mcmh.HashMessage(msg)))
	if err != nil {
		jww.FATAL.Panicf(
			"Failed to save metered message after count update: %s", err)
	}

	msfFormat, err := format.Unmarshal(msg.M)
	if err != nil {
		jww.FATAL.Panicf(
			"Failed to unmarshal message after count update: %s", err)
	}

	ri := &pb.RoundInfo{}
	err = proto.Unmarshal(msg.Ri, ri)
	if err != nil {
		jww.FATAL.Panicf(
			"Failed to unmarshal round info from msg format: %s", err)
	}

	identity := receptionID.EphemeralIdentity{}
	err = json.Unmarshal(msg.Identity, &identity)
	if err != nil {
		jww.FATAL.Panicf(
			"Failed to unmarshal identity from msg format: %s", err)
	}

	return msfFormat, ri, identity, true
}

func (mcmb *MeteredCmixMessageBuffer) Remove(m format.Message, ri *pb.RoundInfo,
	identity receptionID.EphemeralIdentity) {
	mcmb.mb.Succeeded(buildMsg(m, ri, identity))
}

func (mcmb *MeteredCmixMessageBuffer) Failed(m format.Message, ri *pb.RoundInfo,
	identity receptionID.EphemeralIdentity) {
	mcmb.mb.Failed(buildMsg(m, ri, identity))
}

func buildMsg(m format.Message, ri *pb.RoundInfo,
	identity receptionID.EphemeralIdentity) meteredCmixMessage {
	if m.GetPrimeByteLen() == 0 {
		jww.FATAL.Panic(
			"Cannot handle a metered cMix message with a length of 0.")
	}
	riMarshal, err := proto.Marshal(ri)
	if err != nil {
		jww.FATAL.Panicf("Failed to marshal round info: %s", err)
	}

	identityMarshal, err := json.Marshal(&identity)
	if err != nil {
		jww.FATAL.Panicf("Failed to marshal identity: %s", err)
	}

	return meteredCmixMessage{
		M:         m.Marshal(),
		Ri:        riMarshal,
		Identity:  identityMarshal,
		Count:     0,
		Timestamp: time.Unix(0, int64(ri.Timestamps[states.QUEUED])),
	}
}
