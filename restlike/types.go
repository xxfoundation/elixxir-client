////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package restlike

import (
	"github.com/pkg/errors"
	"sync"
)

// URI defines the destination endpoint of a Request
type URI string

// Data provides a generic structure for data sent with a Request or received in a Message
// NOTE: The way this is encoded is up to the implementation. For example, protobuf or JSON
type Data []byte

// Method defines the possible Request types
type Method uint32

// RequestCallback provides the ability to make asynchronous Request
// in order to get the Message later without blocking
type RequestCallback func(*Message)

// Callback serves as an Endpoint function to be called when a Request is received
// Should return the desired response to be sent back to the sender
type Callback func(*Message) *Message

const (
	// Undefined default value
	Undefined Method = iota
	// Get retrieve an existing resource.
	Get
	// Post creates a new resource.
	Post
	// Put updates an existing resource.
	Put
	// Patch partially updates an existing resource.
	Patch
	// Delete a resource.
	Delete
)

// methodStrings is a map of Method values back to their constant names for printing
var methodStrings = map[Method]string{
	Undefined: "undefined",
	Get:       "get",
	Post:      "post",
	Put:       "put",
	Patch:     "patch",
	Delete:    "delete",
}

// String returns the Method as a human-readable name.
func (m Method) String() string {
	if methodStr, ok := methodStrings[m]; ok {
		return methodStr
	}
	return methodStrings[Undefined]
}

// Endpoints represents a map of internal endpoints for a RestServer
type Endpoints struct {
	endpoints map[URI]map[Method]Callback
	sync.RWMutex
}

// Add a new Endpoint
// Returns an error if Endpoint already exists
func (e *Endpoints) Add(path URI, method Method, cb Callback) error {
	e.Lock()
	defer e.Unlock()

	if _, ok := e.endpoints[path]; !ok {
		e.endpoints[path] = make(map[Method]Callback)
	}
	if _, ok := e.endpoints[path][method]; ok {
		return errors.Errorf("unable to RegisterEndpoint: %s/%s already exists", path, method)
	}
	e.endpoints[path][method] = cb
	return nil
}

// Get an Endpoint
// Returns an error if Endpoint does not exist
func (e *Endpoints) Get(path URI, method Method) (Callback, error) {
	e.RLock()
	defer e.RUnlock()

	if _, ok := e.endpoints[path]; !ok {
		return nil, errors.Errorf("unable to locate endpoint: %s", path)
	}
	if _, innerOk := e.endpoints[path][method]; !innerOk {
		return nil, errors.Errorf("unable to locate endpoint: %s/%s", path, method)
	}
	return e.endpoints[path][method], nil
}

// Remove an Endpoint
// Returns an error if Endpoint does not exist
func (e *Endpoints) Remove(path URI, method Method) error {
	if _, err := e.Get(path, method); err != nil {
		return errors.Errorf("unable to UnregisterEndpoint: %s", err.Error())
	}

	e.Lock()
	defer e.Unlock()
	delete(e.endpoints[path], method)
	if len(e.endpoints[path]) == 0 {
		delete(e.endpoints, path)
	}
	return nil
}
