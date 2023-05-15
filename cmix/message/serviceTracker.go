////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"encoding/json"
	"gitlab.com/xx_network/primitives/id"
)

type ServicesTracker func(ServiceList, CompressedServiceList)

// TrackServices adds a service tracker to be triggered when a new service is
// added. Generally used for notification to use this system to identify a
// received message.
func (sm *ServicesManager) TrackServices(tracker ServicesTracker) {
	if tracker == nil {
		return
	}
	sm.Lock()
	defer sm.Unlock()

	sm.trackers = append(sm.trackers, tracker)
}

// triggerServiceTracking triggers the tracking of services. Is it called when a
// service is added or removed.
func (sm *ServicesManager) triggerServiceTracking() {
	if len(sm.trackers) == 0 {
		return
	}
	services := make(ServiceList)
	for uid, tmap := range sm.services {
		tList := make([]Service, 0, len(tmap))
		for _, s := range tmap {
			tList = append(tList, s.Service)
		}
		services[uid] = tList
	}
	cServices := make(CompressedServiceList)
	for uid, tmap := range sm.compressedServices {
		tList := make([]CompressedService, 0, len(tmap))
		for _, s := range tmap {
			tList = append(tList, s.CompressedService)
		}
		cServices[uid] = tList
	}

	for _, callback := range sm.trackers {
		go callback(services, cServices)
	}
}

// The ServiceList holds all services.
type ServiceList map[id.ID][]Service
type CompressedServiceList map[id.ID][]CompressedService

type slMarshaled struct {
	Id                 id.ID
	Services           []Service
	CompressedServices []CompressedService
}

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
