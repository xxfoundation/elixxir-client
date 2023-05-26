package collective

import "sync"

type Notifier interface {
	Register(nc NotifyCallback)
}

type NotifyCallback func(state bool)

type notifier struct {
	toNotify []NotifyCallback
	mux      sync.Mutex
}

func (n *notifier) Register(nc NotifyCallback) {
	n.mux.Lock()
	defer n.mux.Unlock()
	n.toNotify = append(n.toNotify, nc)
}

func (n *notifier) Notify(state bool) {
	n.mux.Lock()
	defer n.mux.Unlock()
	for _, f := range n.toNotify {
		go func(nc NotifyCallback) {
			nc(state)
		}(f)
	}
}
