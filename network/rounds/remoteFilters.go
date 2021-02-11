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
func ValidFilterRange(identity reception.IdentityUse, filters *mixmessages.ClientBlooms) (startIdx int, endIdx int, outOfBounds bool) {
	outOfBounds = false

	firstElementTS := filters.FirstTimestamp

	identityStart := identity.StartValid.UnixNano()
	identityEnd := identity.EndValid.UnixNano()

	startIdx = int((identityStart - firstElementTS)/filters.Period)
	if startIdx < 0{
		startIdx = 0
	}

	if startIdx > len(filters.Filters)-1{
		outOfBounds = true
		return startIdx, endIdx, outOfBounds
	}

	endIdx = int((identityEnd - firstElementTS)/filters.Period)
	if endIdx<0{
		outOfBounds = true
		return startIdx, endIdx, outOfBounds
	}

	if int(endIdx) > len(filters.Filters)-1{
		endIdx = len(filters.Filters)-1
	}

	return startIdx, endIdx, outOfBounds
}
