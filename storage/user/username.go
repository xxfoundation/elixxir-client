////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentUsernameVersion = 0
const usernameKey = "username"

func (u *User) loadUsername() {
	u.usernameMux.Lock()
	obj, err := u.kv.Get(usernameKey, currentUsernameVersion)
	if err == nil {
		u.username = string(obj.Data)
	}
	u.usernameMux.Unlock()
}

func (u *User) SetUsername(username string) error {
	u.usernameMux.Lock()
	defer u.usernameMux.Unlock()
	if u.username != "" {
		return errors.New("Cannot set username when already set")
	}

	obj := &versioned.Object{
		Version:   currentUsernameVersion,
		Timestamp: netTime.Now(),
		Data:      []byte(username),
	}

	err := u.kv.Set(usernameKey, obj)
	if err != nil {
		jww.FATAL.Panicf("Failed to store the username: %s", err)
	}

	u.username = username

	return nil
}

func (u *User) GetUsername() (string, error) {
	u.usernameMux.RLock()
	defer u.usernameMux.RUnlock()
	if u.username == "" {
		return "", errors.New("no username set")
	}
	return u.username, nil
}
