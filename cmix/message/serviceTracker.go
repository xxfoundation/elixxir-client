////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"encoding/json"

	"gitlab.com/elixxir/crypto/sih"
	"gitlab.com/xx_network/primitives/id"
)

// ServicesTracker returns the current [ServiceList] and [CompressedServiceList]
// stored in the [ServicesManager].
type ServicesTracker func(ServiceList, CompressedServiceList)

// The ServiceList holds all Service keyed on their user ID.
type ServiceList map[id.ID][]Service

// The CompressedServiceList holds all CompressedService keyed on their user ID.
type CompressedServiceList map[id.ID][]CompressedService

// TrackServices registers a callback that is called every time a service is
// added or removed. It is also called once when registered. The callback
// receives the new service lists every time they are modified. Callbacks only
// occur when the network follower is running. Multiple
// [message.ServicesTracker] can be registered.
//
// Generally, this is used by notification to identify a received message.
func (sm *ServicesManager) TrackServices(tracker ServicesTracker) {
	if tracker == nil {
		return
	}
	sm.Lock()
	defer sm.Unlock()

	sm.trackers = append(sm.trackers, tracker)

	// Call the callback
	go tracker(sm.sl.DeepCopy(), sm.csl.DeepCopy())
}

// GetServices returns the current list of registered services and compressed
// services. This returns the same lists as the last lists provided to trackers
// registered with [ServicesManager.TrackServices].
func (sm *ServicesManager) GetServices() (ServiceList, CompressedServiceList) {
	sm.Lock()
	defer sm.Unlock()

	return sm.sl, sm.csl
}

// triggerServiceTracking triggers the tracking of services. Is it called when a
// service is added or removed.
func (sm *ServicesManager) triggerServiceTracking() {
	if len(sm.trackers) == 0 {
		return
	}

	// Generate and update service lists
	sm.sl = makeServiceList(sm.services)
	sm.csl = makeCompressedServiceList(sm.compressedServices)

	for _, callback := range sm.trackers {
		go callback(sm.sl.DeepCopy(), sm.csl.DeepCopy())
	}
}

// makeServiceList returns the map of services as a ServiceList.
func makeServiceList(services map[id.ID]map[sih.Preimage]service) ServiceList {
	sl := make(ServiceList, len(services))
	for uid, sMap := range services {
		sList := make([]Service, 0, len(sMap))
		for _, s := range sMap {
			sList = append(sList, s.Service)
		}
		sl[uid] = sList
	}
	return sl
}

// makeServiceList returns the map of compressedServices as a
// CompressedServiceList.
func makeCompressedServiceList(
	cServices map[id.ID]map[sih.Preimage]compressedService) CompressedServiceList {
	csl := make(CompressedServiceList, len(cServices))
	for uid, sMap := range cServices {
		sList := make([]CompressedService, 0, len(sMap))
		for _, s := range sMap {
			sList = append(sList, s.CompressedService)
		}
		csl[uid] = sList
	}
	return csl
}

// DeepCopy creates a copy of all public fields of the [ServiceList].
func (sl ServiceList) DeepCopy() ServiceList {
	newSl := make(ServiceList, len(sl))

	for uid, l := range sl {
		newSl[uid] = make([]Service, len(l))
		for i, s := range l {
			newService := Service{
				Identifier: append([]byte{}, s.Identifier...),
				Tag:        s.Tag,
				Metadata:   append([]byte{}, s.Metadata...),
			}

			newSl[uid][i] = newService
		}
	}

	return newSl
}

// DeepCopy creates a copy of all public fields of the [CompressedServiceList].
func (csl CompressedServiceList) DeepCopy() CompressedServiceList {
	newCsl := make(CompressedServiceList, len(csl))

	for uid, l := range csl {
		newCsl[uid] = make([]CompressedService, len(l))
		for i, s := range l {
			newService := CompressedService{
				Identifier: append([]byte{}, s.Identifier...),
				Tags:       append([]string{}, s.Tags...),
				Metadata:   append([]byte{}, s.Metadata...),
			}

			newCsl[uid][i] = newService
		}
	}

	return newCsl
}

// slMarshaled contains a user ID and the services list in an object that can
// be JSON marshalled and unmarshalled.
type slMarshaled struct {
	Id                 id.ID
	Services           []Service
	CompressedServices []CompressedService
}

// MarshalJSON marshals the ServiceList into valid JSON. This function adheres
// to the [json.Marshaler] interface.
func (sl ServiceList) MarshalJSON() ([]byte, error) {
	slList := make([]slMarshaled, 0, len(sl))
	for uid, s := range sl {
		slList = append(slList, slMarshaled{
			Id:       uid,
			Services: s,
		})
	}
	return json.Marshal(&slList)
}

// UnmarshalJSON unmarshalls JSON into the [ServiceList]. This function adheres to
// the [json.Unmarshaler] interface.
//
// Note that this function does not transfer the internal RNG. Use
// NewCipherFromJSON to properly reconstruct a cipher from JSON.
// UnmarshalJSON adheres to the json.Unmarshaler interface.
func (sl ServiceList) UnmarshalJSON(b []byte) error {
	slList := make([]slMarshaled, 0)
	if err := json.Unmarshal(b, &slList); err != nil {
		return err
	}
	for _, s := range slList {
		sl[s.Id] = s.Services
	}
	return nil
}
