package rounds

import (
	"fmt"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
)

type Round struct {
	// ID of the round. IDs are sequential and monotonic.
	ID id.Round

	// State is the last known state of the round. Possible states are:
	//   PENDING - not started yet
	//   PRECOMPUTING - In the process of preparing to process messages
	//   STANDBY - Completed precomputing but not yet scheduled to run
	//	 QUEUED - Scheduled to run at a set time
	//	 REALTIME - Running, actively handing messages
	//	 COMPLETED - Successfully deleted messages
	//	 FAILED - Failed to deliver messages
	State states.Round

	// Topology contains the list of nodes in the round.
	Topology *connect.Circuit

	// Timestamps of all events that have occurred in the round (see the above
	// states).
	// The QUEUED state's timestamp is different; it denotes when Realtime
	// was/is scheduled to start, not whe the QUEUED state is entered.
	Timestamps map[states.Round]time.Time

	// Errors that occurred in the round. Will only be present in the failed
	// state.
	Errors []RoundError

	/* Properties */

	// BatchSize is the max number of messages the round can process.
	BatchSize uint32

	// AddressSpaceSize is the ephemeral address space size used in the round.
	AddressSpaceSize uint8

	// UpdateID is a monotonic counter between all round updates denoting when
	// this updated occurred in the queue.
	UpdateID uint64

	// Raw is raw round data, including signatures.
	Raw *pb.RoundInfo
}

type RoundError struct {
	NodeID *id.ID
	Error  string
}

// MakeRound builds an accessible round object from a RoundInfo protobuf.
func MakeRound(ri *pb.RoundInfo) Round {
	// Build the timestamps map
	timestamps := make(map[states.Round]time.Time)

	for i := range ri.Timestamps {
		if ri.Timestamps[i] != 0 {
			timestamps[states.Round(i)] =
				time.Unix(0, int64(ri.Timestamps[i]))
		}
	}

	// Build the input to the topology
	nodes := make([]*id.ID, len(ri.Topology))
	for i := range ri.Topology {
		newNodeID := id.ID{}
		copy(newNodeID[:], ri.Topology[i])
		nodes[i] = &newNodeID
	}

	// Build the errors
	errs := make([]RoundError, len(ri.Errors))
	for i := range ri.Errors {
		errNodeID := id.ID{}
		copy(errNodeID[:], ri.Errors[i].NodeId)
		errs[i] = RoundError{
			NodeID: &errNodeID,
			Error:  ri.Errors[i].Error,
		}
	}

	return Round{
		ID:               id.Round(ri.ID),
		State:            states.Round(ri.State),
		Topology:         connect.NewCircuit(nodes),
		Timestamps:       timestamps,
		Errors:           errs,
		BatchSize:        ri.BatchSize,
		AddressSpaceSize: uint8(ri.AddressSpaceSize),
		UpdateID:         ri.UpdateID,
		Raw:              ri,
	}
}

// GetEndTimestamp returns the timestamp of the last known event, which is
// generally the state unless in queued, which stores the next event.
func (r Round) GetEndTimestamp() time.Time {
	switch r.State {
	case states.PENDING:
		return r.Timestamps[states.PENDING]
	case states.PRECOMPUTING:
		return r.Timestamps[states.PRECOMPUTING]
	case states.STANDBY:
		return r.Timestamps[states.STANDBY]
	case states.QUEUED:
		return r.Timestamps[states.QUEUED]
	case states.REALTIME:
		return r.Timestamps[states.REALTIME]
	case states.COMPLETED:
		return r.Timestamps[states.COMPLETED]
	case states.FAILED:
		return r.Timestamps[states.FAILED]
	default:
		jww.FATAL.Panicf("Could not get final timestamp of round, "+
			"invalid state: %s", r.State)
	}

	// Unreachable
	return time.Time{}
}

// String prints a formatted version of the client error string
func (re *RoundError) String() string {
	return fmt.Sprintf("ClientError(ClientID: %s, Err: %s)",
		re.NodeID, re.Error)
}
