package notifications

import (
	"gitlab.com/xx_network/primitives/id"
)

// SetMaxState sets the maximum functional state of any identity
// downstream moduals will be told to clamp any state greater than maxState
// down to maxState. Depending on UX requirements, they may still show the
// state in an altered manner, for example greying out a description.
// This is designed so when the state is raised, the old configs are
// maintained.
// This will unregister / re-register with the push server when leaving or
// entering the Push maxState.
// The default maxState is Push
// will return an error if the maxState isnt a valid state
func (m *manager) SetMaxState(maxState NotificationState) error {
	if err := maxState.IsValid(); err != nil {
		return err
	}
	m.mux.Lock()
	defer m.mux.Unlock()

	if maxState < Push && m.maxState == Push {
		//unregister all
		pushList := m.getPushed()
		if err := m.unregisterNotification(pushList); err != nil {
			return err
		}
	} else if maxState == Push && m.maxState != Push {
		pushList := m.getPushed()
		if err := m.registerNotification(pushList); err != nil {
			return err
		}
	}

	m.setMaxStateUnsafe(maxState)
	return nil
}

// GetMaxState returns the current MaxState
func (m *manager) GetMaxState() NotificationState {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return m.maxState
}

// getPushed finds all IDs in the "Push" state and returns them
// must be called under the lock in read or write mode
func (m *manager) getPushed() []*id.ID {
	pushList := make([]*id.ID, 0, len(m.notifications))
	for nid, reg := range m.notifications {
		localNid := nid
		if reg.State.Status == Push {
			pushList = append(pushList, &localNid)
		}
	}
	return pushList
}
