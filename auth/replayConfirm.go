////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package auth

import "gitlab.com/xx_network/primitives/id"

// ReplayConfirm is used to resend a confirm
func (s *state) ReplayConfirm(partner *id.ID) (id.Round, error) {

	confirmPayload, mac, keyfp, err := s.store.LoadConfirmation(partner)
	if err != nil {
		return 0, err
	}

	rid, err := sendAuthConfirm(s.net, partner, keyfp,
		confirmPayload, mac, s.event, s.params.ResetConfirmTag)

	return rid, err
}
