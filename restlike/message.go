////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package restlike

import (
	"github.com/pkg/errors"
)

// Message are used for sending to and receiving from a RestServer
type Message interface {
	Content() Data
	Headers() Param
	Method() Method
	URI() URI
	Error() error
}

// message implements the Message interface using JSON
type message struct {
	content Data
	headers Param
	method  Method
	uri     URI
	err     string
}

func (m message) Content() Data {
	return m.content
}

func (m message) Headers() Param {
	return m.headers
}

func (m message) Method() Method {
	return m.method
}

func (m message) URI() URI {
	return m.uri
}

func (m message) Error() error {
	if len(m.err) == 0 {
		return nil
	}
	return errors.New(m.err)
}
