////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/broadcast"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"sync"
)

// Error messages
const (
	noProcessorChannelErr = "no processors found for channel %s"
	noProcessorTagErr     = "no processor %s found for channel %s"
)

// processorTag represents a tag in the processor list to retrieve a processor
// for a channel.
type processorTag uint8

const (
	userProcessor  processorTag = iota
	adminProcessor processorTag = iota
)

// processorList contains the list of processors for each channel.
type processorList struct {
	list map[id.ID]map[processorTag]broadcast.Processor
	mux  sync.RWMutex
}

// newProcessorList initialises an empty processorList.
func newProcessorList() *processorList {
	return &processorList{
		list: make(map[id.ID]map[processorTag]broadcast.Processor),
	}
}

// addProcessor adds the broadcast.Processor for the given tag to the channel.
// This overwrites any previously saved processor at the same location. This
// function is thread safe.
func (pl *processorList) addProcessor(
	channelID *id.ID, tag processorTag, p broadcast.Processor) {
	pl.mux.Lock()
	defer pl.mux.Unlock()
	if _, exists := pl.list[*channelID]; !exists {
		pl.list[*channelID] = make(map[processorTag]broadcast.Processor, 1)
	}
	pl.list[*channelID][tag] = p
}

// removeProcessors removes all registered processors for the channel. Use this
// when leaving a channel. This function is thread safe.
func (pl *processorList) removeProcessors(channelID *id.ID) {
	pl.mux.Lock()
	defer pl.mux.Unlock()
	delete(pl.list, *channelID)
}

// getProcessor returns the broadcast.Processor for the given tag in the
// channel. Returns an error if the processor does not exist. This function is
// thread safe.
func (pl *processorList) getProcessor(
	channelID *id.ID, tag processorTag) (broadcast.Processor, error) {
	pl.mux.RLock()
	defer pl.mux.RUnlock()

	if processors, exists := pl.list[*channelID]; !exists {
		return nil, errors.Errorf(noProcessorChannelErr, channelID)
	} else if p, exists2 := processors[tag]; !exists2 {
		return nil, errors.Errorf(noProcessorTagErr, tag, channelID)
	} else {
		return p, nil
	}
}

// String prints a human-readable form of the processorTag for logging and
// debugging. This function adheres to the fmt.Stringer interface.
func (pt processorTag) String() string {
	switch pt {
	case userProcessor:
		return "userProcessor"
	case adminProcessor:
		return "adminProcessor"
	default:
		return "INVALID PROCESSOR TAG " + strconv.Itoa(int(pt))
	}
}
