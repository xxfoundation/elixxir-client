////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	jww "github.com/spf13/jwalterweatherman"
	bloom "gitlab.com/elixxir/bloomfilter"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
)

const BloomFilterSize = 648 // In Bits
const BloomFilterHashes = 10

func NewRemoteFilter(data *mixmessages.ClientBloom) *RemoteFilter {
	return &RemoteFilter{
		data: data,
	}
}

type RemoteFilter struct {
	data   *mixmessages.ClientBloom
	filter *bloom.Bloom
}

func (rf *RemoteFilter) GetFilter() *bloom.Bloom {

	if rf.filter == nil {
		var err error
		rf.filter, _ = bloom.InitByParameters(BloomFilterSize,
			BloomFilterHashes)
		err = rf.filter.UnmarshalBinary(rf.data.Filter)
		if err != nil {
			jww.FATAL.Panicf(
				"Failed to properly unmarshal the bloom filter: %+v", err)
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
