////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

// ConnectionList is a list of all connections.
type ConnectionList struct {
	list map[id.ID]Connection
	p    ConnectionListParams
	mux  sync.Mutex
}

// NewConnectionList initialises an empty ConnectionList.
func NewConnectionList(p ConnectionListParams) *ConnectionList {
	return &ConnectionList{
		list: make(map[id.ID]Connection),
		p:    p,
	}
}

// Add adds the connection to the list.
func (cl *ConnectionList) Add(c Connection) {
	cl.mux.Lock()
	defer cl.mux.Unlock()

	cl.list[*c.GetPartner().PartnerId()] = c
}

// CleanupThread runs the loop that runs the cleanup processes periodically.
func (cl *ConnectionList) CleanupThread() (stoppable.Stoppable, error) {
	stop := stoppable.NewSingle("StaleConnectionCleanup")

	go func() {
		jww.INFO.Printf("Starting stale connection cleanup thread to delete "+
			"connections older than %s. Running every %s.",
			cl.p.MaxAge, cl.p.CleanupPeriod)
		ticker := time.NewTicker(cl.p.CleanupPeriod)
		for {
			select {
			case <-stop.Quit():
				jww.INFO.Print(
					"Stopping connection cleanup thread: stoppable triggered")
				ticker.Stop()
				stop.ToStopped()
			case <-ticker.C:
				jww.DEBUG.Print("Starting connection cleanup.")
				cl.Cleanup()
			}
		}
	}()

	return stop, nil
}

// Cleanup disconnects all connections that have been stale for longer than the
// max allowed time.
func (cl *ConnectionList) Cleanup() {
	cl.mux.Lock()
	defer cl.mux.Unlock()

	for partnerID, c := range cl.list {
		lastUse := c.LastUse()
		timeSinceLastUse := netTime.Since(lastUse)
		if timeSinceLastUse > cl.p.MaxAge {
			err := c.Close()
			if err != nil {
				jww.ERROR.Printf(
					"Could not close connection with partner %s: %+v",
					partnerID, err)
			}
			delete(cl.list, partnerID)

			jww.INFO.Printf("Deleted stale connection for partner %s. "+
				"Last use was %s ago (%s)",
				&partnerID, timeSinceLastUse, lastUse.Local())
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Parameters                                                                 //
////////////////////////////////////////////////////////////////////////////////

// Default values.
const (
	cleanupPeriodDefault = 5 * time.Minute
	maxAgeDefault        = 30 * time.Minute
)

// ConnectionListParams are the parameters used for the ConnectionList.
type ConnectionListParams struct {
	// CleanupPeriod is the duration between when cleanups occur.
	CleanupPeriod time.Duration

	// MaxAge is the maximum age of an unused connection before it is deleted.
	MaxAge time.Duration
}

// DefaultConnectionListParams returns a ConnectionListParams filled with
// default values.
func DefaultConnectionListParams() ConnectionListParams {
	return ConnectionListParams{
		CleanupPeriod: cleanupPeriodDefault,
		MaxAge:        maxAgeDefault,
	}
}
