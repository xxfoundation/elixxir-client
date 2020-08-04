package storage

import "gitlab.com/elixxir/ekv"

type Session struct {
	kv *VersionedKV
}

func Init(baseDir, password string) (*Session, error) {
	fs, err := ekv.NewFilestore(baseDir, password)
	var s *Session
	if err == nil {
		s = &Session{
			kv: NewVersionedKV(fs),
		}
	}

	return s, err
}

func (s *Session) Get(key string) (*VersionedObject, error) {
	return s.kv.Get(key)
}

func (s *Session) Set(key string, object *VersionedObject) error {
	return s.kv.Set(key, object)
}
