package notifications

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math"
	"time"
)

// RunHandler starts a loop which receives pendingRegistration objects over the manager regChan
// and handles registration/deletion asynchronously from the main thread.
func (m *manager) RunHandler(name string) *stoppable.Single {
	stop := stoppable.NewSingle(name)
	go func() {
		for {
			select {
			case <-stop.Quit():
				jww.INFO.Println("Handler thread received stop signal")
				stop.ToStopped()
				return
			case reg := <-m.regChan:
				jww.INFO.Printf("Handling remote registration for %s", reg.nid.String())
				// TODO do we want these to happen in parallel?
				m.handlePendingRegistrationUpdate(reg)
			}
		}
	}()
	return stop
}

func (m *manager) handlePendingRegistrationUpdate(reg pendingRegistration) {
	var numAttempts, sleepTime, maxSleep float64
	numAttempts = 0
	sleepTime = 1
	maxSleep = 512
	for done := false; !done; {
		if numAttempts > 0 {
			sleepFor := math.Min(math.Pow(sleepTime, numAttempts), maxSleep)
			time.Sleep(time.Second * time.Duration(sleepFor))
		}
		numAttempts++

		if reg.r.PendingDeletion {
			err := m.handleDeletion(reg)
			if err != nil {
				jww.ERROR.Printf("Error handling deletion: %+v", err)
			}
		} else {
			err := m.handleRegistration(reg)
			if err != nil {
				jww.ERROR.Printf("Error handling registration: %+v", err)
			}
		}
		done = true
	}
}

func (m *manager) handleRegistration(reg pendingRegistration) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	currentReg, exists := m.notifications[*reg.nid]
	if exists {
		if currentReg.Group != reg.r.Group {
			return errors.Errorf("cannot change the group of a notification " +
				"registration")
		}
		if currentReg.Confirmed {
			return nil
		}
		if currentReg.Status != reg.r.Status ||
			!bytes.Equal(reg.r.Metadata, currentReg.Metadata) {
			return errors.Errorf("Registration found by handler does not match registration in storage")
		}
	} else {
		jww.WARN.Println("Should not reach handleregistration with no registration in ram")
	}

	// register with remote
	if reg.r.Status == Push {
		if err := m.registerNotification([]*id.ID{reg.nid}); err != nil {
			return err
		}
	} else if reg.r.Status != Push {
		if err := m.unregisterNotification([]*id.ID{reg.nid}); err != nil {
			return err
		}
	}

	ts := netTime.Now()

	updatedRegistration := registration{
		Group:     reg.r.Group,
		Confirmed: true,
		State: State{
			Metadata: reg.r.State.Metadata,
			Status:   reg.r.State.Status,
		},
	}

	err := m.storeRegistration(reg.nid, updatedRegistration, ts)
	if err != nil {
		return err
	}

	// update in ram storage
	m.upsertNotificationUnsafeRAM(reg.nid, updatedRegistration)

	// call the callback if it exists to notify that an update exists
	if cb, cbExists := m.callbacks[reg.r.Group]; cbExists {
		// can be nil if the last element was deleted
		g, _ := m.group[reg.r.Group]
		var created, updated []*id.ID
		updated = []*id.ID{reg.nid}
		go cb(g.DeepCopy(), created, updated, nil, m.maxState)
	}
	return nil
}

func (m *manager) handleDeletion(reg pendingRegistration) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if reg.r.Status == Push {
		if err := m.unregisterNotification([]*id.ID{reg.nid}); err != nil {
			return err
		}
	}

	elementName := makeElementName(reg.nid)

	_, err := m.remote.DeleteMapElement(notificationsMap, elementName,
		notificationsMapVersion)
	if err != nil {
		return err
	}
	group := m.deleteNotificationUnsafeRAM(reg.nid)

	// call the callback if it exists to notify that an update exists
	if cb, cbExists := m.callbacks[group]; cbExists {
		// can be nil if the last element was deleted
		g, _ := m.group[group]
		go cb(g.DeepCopy(), nil, nil, []*id.ID{reg.nid}, m.maxState)
	}

	return nil
}
