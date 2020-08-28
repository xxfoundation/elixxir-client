package user

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"time"
)

const currentUsernameVersion = 0
const usernameKey = "username"

func (u *User) loadUsername() {
	u.usernameMux.Lock()
	obj, err := u.kv.Get(usernameKey)
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
		Timestamp: time.Now(),
		Data:      []byte(username),
	}

	err := u.kv.Set(usernameKey, obj)
	if err != nil {
		return errors.WithMessage(err, "Failed to store the username")
	}

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
