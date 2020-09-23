package storage

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/client/vendor/github.com/pkg/errors"
	"time"
)

const regCodeKey = "regCode"
const regCodeVersion = 0

// SetNDF stores a network definition json file
func (s *Session) SetRegCode(regCode string) error {
	return s.Set(regCodeKey,
		&versioned.Object{
			Version:   regCodeVersion,
			Data:      []byte(regCode),
			Timestamp: time.Now(),
		})
}

// Returns the stored network definition json file
func (s *Session) GetRegCode() (string, error) {
	regCode, err := s.Get(regCodeKey)
	if err != nil {
		return "", errors.WithMessage(err, "Failed to load the regcode")
	}
	return string(regCode.Data), nil
}
