///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package storage

import "gitlab.com/elixxir/client/interfaces/user"

func (s *Session) GetUser() user.User {
	s.mux.RLock()
	defer s.mux.RUnlock()
	ci := s.user.GetCryptographicIdentity()
	return user.User{
		TransmissionID:        ci.GetTransmissionID().DeepCopy(),
		TransmissionSalt:      copySlice(ci.GetTransmissionSalt()),
		TransmissionRSA:       ci.GetTransmissionRSA(),
		ReceptionID:           ci.GetReceptionID().DeepCopy(),
		RegistrationTimestamp: s.user.GetRegistrationTimestamp().UnixNano(),
		ReceptionSalt:         copySlice(ci.GetReceptionSalt()),
		ReceptionRSA:          ci.GetReceptionRSA(),
		Precanned:             ci.IsPrecanned(),
		E2eDhPrivateKey:       s.e2e.GetDHPrivateKey().DeepCopy(),
		E2eDhPublicKey:        s.e2e.GetDHPublicKey().DeepCopy(),
	}

}

func copySlice(s []byte) []byte {
	n := make([]byte, len(s))
	copy(n, s)
	return n
}
