package auth

import "gitlab.com/xx_network/primitives/id"

//todo implement replay confirm
func (s *state) ReplayConfirm(partner *id.ID) (id.Round, error) {
	return 0, nil
}
