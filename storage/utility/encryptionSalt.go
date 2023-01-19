package utility

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
)

// Storage constats
const (
	saltKey     = "encryptionSalt"
	saltVersion = 0
	saltPrefix  = "encryptionSaltPrefix"
)

// saltSize is the defined size in bytes of the salt generated in
// newSalt.
const saltSize = 32

// NewOrLoadSalt will attempt to find a stored salt if one exists.
// If one does not exist in storage, a new one will be generated. The newly
// generated salt will be stored.
func NewOrLoadSalt(kv *versioned.KV, stream io.Reader) ([]byte, error) {
	kv = kv.Prefix(saltPrefix)
	salt, err := loadSalt(kv)
	if err != nil {
		jww.WARN.Printf("Failed to load salt, generating new one...")
		salt, err = newSalt(kv, stream)
	}

	return salt, err
}

// loadSalt is a helper function which attempts to load a stored salt from
// memory.
func loadSalt(kv *versioned.KV) ([]byte, error) {
	obj, err := kv.Get(saltKey, saltVersion)
	if err != nil {
		return nil, err
	}

	return obj.Data, nil
}

// newSalt generates a new random salt. This salt is stored and returned
// to the caller.
func newSalt(kv *versioned.KV, stream io.Reader) ([]byte, error) {
	// Generate a new salt
	salt := make([]byte, saltSize)
	_, err := stream.Read(salt)
	if err != nil {
		return nil, err
	}

	// Store salt in storage
	obj := &versioned.Object{
		Version:   saltVersion,
		Timestamp: netTime.Now(),
		Data:      salt,
	}

	err = kv.Set(saltKey, obj)
	if err != nil {
		return nil, err
	}

	return salt, nil
}