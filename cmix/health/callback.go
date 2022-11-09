package health

import "sync"

type trackerCallback struct {
	funcs   map[uint64]func(isHealthy bool)
	funcsID uint64

	mux sync.RWMutex
}

func initTrackerCallback() *trackerCallback {
	return &trackerCallback{
		funcs:   map[uint64]func(isHealthy bool){},
		funcsID: 0,
	}
}

// addHealthCallback adds a function to the list of tracker functions such that
// each function can be run after network changes. Returns a unique ID for the
// function.
func (t *trackerCallback) addHealthCallback(f func(isHealthy bool), health bool) uint64 {
	var currentID uint64

	t.mux.Lock()
	t.funcs[t.funcsID] = f
	currentID = t.funcsID
	t.funcsID++
	t.mux.Unlock()

	go f(health)

	return currentID
}

// RemoveHealthCallback removes the function with the given ID from the list of
// tracker functions so that it will no longer be run.
func (t *trackerCallback) RemoveHealthCallback(chanID uint64) {
	t.mux.Lock()
	delete(t.funcs, chanID)
	t.mux.Unlock()
}

// callback calls every function with the new health state
func (t *trackerCallback) callback(health bool) {
	t.mux.Lock()
	defer t.mux.Unlock()

	// Run all listening functions
	for _, f := range t.funcs {
		go f(health)
	}
}
