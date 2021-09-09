package storage

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	mnemonicKvKey     = "mnemonic"
	mnemonicKvVersion = 0
	mnemonicPath      = "/.recovery"
)

func (s *Session) SaveMnemonicInformation(data []byte) error {
	s.mnemonicMux.Lock()
	defer s.mnemonicMux.Unlock()

	vo := &versioned.Object{
		Version:   mnemonicKvVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return s.mnemonicKV.Set(mnemonicKvKey, mnemonicKvVersion, vo)
}

func (s *Session) LoadMnemonicInformation() ([]byte, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	vo, err := s.mnemonicKv.Get(mnemonicKvKey, mnemonicKvVersion)
	if err != nil {
		return nil, err
	}

	return vo.Data, err
}
