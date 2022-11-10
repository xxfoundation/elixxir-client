////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"bytes"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

const (
	uncheckedRoundVersion = 0
	roundInfoVersion      = 0
	uncheckedRoundPrefix  = "uncheckedRoundPrefix"
	roundKeyPrefix        = "roundInfo:"

	// Key to store rounds
	uncheckedRoundKey = "uncheckRounds"

	// Housekeeping constant (used for serializing uint64 ie id.Round)
	uint64Size = 8

	// Maximum checks that can be performed on a round. Intended so that a round
	// is checked no more than 1 week approximately (network/pickup.cappedTries + 7)
	maxChecks = 14
)

// Identity contains round identity information used in message retrieval.
// Derived from reception.Identity saving data needed for message retrieval.
type Identity struct {
	EpdId  ephemeral.Id
	Source *id.ID
}

// UncheckedRound contains rounds that failed on message retrieval. These rounds
// are stored for retry of message retrieval.
type UncheckedRound struct {
	Info *pb.RoundInfo
	Id   id.Round

	Identity
	// Timestamp in which round has last been checked
	LastCheck time.Time
	// Number of times a round has been checked
	NumChecks uint64

	storageUpToDate bool
	beingChecked    bool
}

// marshal serializes UncheckedRound r into a byte slice.
func (r UncheckedRound) marshal(kv *versioned.KV) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	// Store teh round info
	if r.Info != nil && !r.storageUpToDate {
		if err := storeRoundInfo(kv, r.Info, r.Source, r.EpdId); err != nil {
			return nil, errors.WithMessagef(err,
				"failed to marshal unchecked rounds")
		}
		r.storageUpToDate = true
	}

	// Marshal the round ID
	b := make([]byte, uint64Size)
	binary.LittleEndian.PutUint64(b, uint64(r.Id))
	buf.Write(b)

	// Write the round identity info
	buf.Write(r.Identity.EpdId[:])
	if r.Source != nil {
		buf.Write(r.Identity.Source.Marshal())
	} else {
		buf.Write(make([]byte, id.ArrIDLen))
	}

	// Write the time stamp bytes
	tsBytes, err := r.LastCheck.MarshalBinary()
	if err != nil {
		return nil, errors.WithMessage(err, "Could not marshal timestamp ")
	}
	b = make([]byte, uint64Size)
	binary.LittleEndian.PutUint64(b, uint64(len(tsBytes)))
	buf.Write(b)
	buf.Write(tsBytes)

	// Write the number of tries for this round
	b = make([]byte, uint64Size)
	binary.LittleEndian.PutUint64(b, r.NumChecks)
	buf.Write(b)

	return buf.Bytes(), nil
}

// unmarshal deserializes round data from buff into UncheckedRound r.
func (r *UncheckedRound) unmarshal(kv *versioned.KV, buff *bytes.Buffer) error {
	// Deserialize the roundInfo
	r.Id = id.Round(binary.LittleEndian.Uint64(buff.Next(uint64Size)))

	// Deserialize the round identity information
	copy(r.EpdId[:], buff.Next(uint64Size))

	sourceId, err := id.Unmarshal(buff.Next(id.ArrIDLen))
	if err != nil {
		return errors.WithMessagef(err,
			"Failed to unmarshal round Identity source of round %d", r.Id)
	}

	r.Source = sourceId

	// Deserialize the timestamp bytes
	timestampLen := binary.LittleEndian.Uint64(buff.Next(uint64Size))
	tsByes := buff.Next(int(timestampLen))
	if err = r.LastCheck.UnmarshalBinary(tsByes); err != nil {
		return errors.WithMessagef(err,
			"Failed to unmarshal round timestamp of %d", r.Id)
	}

	r.NumChecks = binary.LittleEndian.Uint64(buff.Next(uint64Size))

	r.Info, _ = loadRoundInfo(kv, r.Id, r.Source, r.EpdId)
	r.storageUpToDate = true

	return nil
}

func storeRoundInfo(kv *versioned.KV, info *pb.RoundInfo, recipient *id.ID,
	ephID ephemeral.Id) error {
	now := netTime.Now()

	data, err := proto.Marshal(info)
	if err != nil {
		return errors.WithMessagef(err,
			"Failed to store individual unchecked round")
	}

	obj := versioned.Object{
		Version:   roundInfoVersion,
		Timestamp: now,
		Data:      data,
	}

	return kv.Set(
		roundKey(id.Round(info.ID), recipient, ephID), &obj)
}

func loadRoundInfo(kv *versioned.KV, id id.Round, recipient *id.ID,
	ephID ephemeral.Id) (*pb.RoundInfo, error) {

	vo, err := kv.Get(roundKey(id, recipient, ephID), roundInfoVersion)
	if err != nil {
		return nil, err
	}

	ri := &pb.RoundInfo{}
	if err = proto.Unmarshal(vo.Data, ri); err != nil {
		return nil, errors.WithMessagef(err, "Failed to unmarshal roundInfo")
	}

	return ri, nil
}

func deleteRoundInfo(kv *versioned.KV, id id.Round, recipient *id.ID,
	ephID ephemeral.Id) error {
	return kv.Delete(roundKey(id, recipient, ephID), roundInfoVersion)
}

func roundKey(roundID id.Round, recipient *id.ID, ephID ephemeral.Id) string {
	return roundKeyPrefix + newRoundIdentity(roundID, recipient, ephID).String()
}
