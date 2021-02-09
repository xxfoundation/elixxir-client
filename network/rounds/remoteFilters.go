package rounds

import (
	bloom "gitlab.com/elixxir/bloomfilter"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage/reception"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

func NewRemoteFilter(data *mixmessages.ClientBloom) *RemoteFilter {
	return &RemoteFilter{
		data: data,
	}
}

type RemoteFilter struct {
	data   *mixmessages.ClientBloom
	filter *bloom.Ring
}

func (rf *RemoteFilter) GetFilter() *bloom.Ring {

	if rf.filter == nil {
		var err error
		rf.filter, err = bloom.InitByParameters(interfaces.BloomFilterSize,
			interfaces.BloomFilterHashes)
		if err != nil {
			return nil
		}
	}
	return rf.filter
}

func (rf *RemoteFilter) FirstRound() id.Round {
	return id.Round(rf.data.FirstRound)
}

func (rf *RemoteFilter) LastRound() id.Round {
	return id.Round(rf.data.FirstRound + uint64(rf.data.RoundRange))
}

// ValidFilterRange calculates which of the returned filters are valid for the identity
func ValidFilterRange(identity reception.IdentityUse, filters *mixmessages.ClientBlooms) (start int, end int) {
	firstFilterStart := time.Unix(0, filters.FirstTimestamp)
	filterDelta := time.Duration(filters.Period)

	deltaFromStart := int(identity.StartValid.Sub(firstFilterStart) / filterDelta)
	deltaFromEnd := int((identity.EndValid.Sub(firstFilterStart) + filterDelta - 1) / filterDelta)
	if deltaFromEnd > (len(filters.Filters) - 1) {
		deltaFromEnd = len(filters.Filters)
	}
	return deltaFromStart, deltaFromEnd + 1
}
