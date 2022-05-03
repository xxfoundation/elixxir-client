////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package restlike

import "github.com/pkg/errors"

// URI defines the destination endpoint of a Request
type URI string

// Data provides a generic structure for data sent with a Request or received in a Message
// NOTE: The way this is encoded is up to the implementation. For example, protobuf or JSON
type Data string

// Method defines the possible Request types
type Method uint8

// Callback provides the ability to make asynchronous Request
// in order to get the Message later without blocking
type Callback func(Message)

// Param allows different configurations for each Request
// that will be specified in the Request header
type Param struct {
	// Version allows for endpoints to be backwards-compatible
	// and handle different formats of the same Request
	Version uint

	// Headers allows for custom headers to be included with a Request
	Headers Data
}

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
type Endpoints map[URI]map[Method]Callback

// Add a new Endpoint
// Returns an error if Endpoint already exists
func (e Endpoints) Add(path URI, method Method, cb Callback) error {
	if _, ok := e[path]; !ok {
		e[path] = make(map[Method]Callback)
	}
	if _, ok := e[path][method]; ok {
		return errors.Errorf("unable to RegisterEndpoint: %s/%s already exists", path, method)
	}
	e[path][method] = cb
	return nil
}

// Get an Endpoint
// Returns an error if Endpoint does not exist
func (e Endpoints) Get(path URI, method Method) (Callback, error) {
	if _, ok := e[path]; !ok {
		return nil, errors.Errorf("unable to locate endpoint: %s", path)
	}
	if _, innerOk := e[path][method]; !innerOk {
		return nil, errors.Errorf("unable to locate endpoint: %s/%s", path, method)
	}
	return e[path][method], nil
}

// Remove an Endpoint
// Returns an error if Endpoint does not exist
func (e Endpoints) Remove(path URI, method Method) error {
	if _, err := e.Get(path, method); err != nil {
		return errors.Errorf("unable to UnregisterEndpoint: %s", err.Error())
	}
	delete(e[path], method)
	if len(e[path]) == 0 {
		delete(e, path)
	}
	return nil
}
