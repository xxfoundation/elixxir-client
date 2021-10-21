package bindings

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/xx_network/primitives/id"
)

type PreimageNotification interface {
	Notify(identity []byte, deleted bool)
}

func (c *Client) RegisterPreimageCallback(identity []byte, pin PreimageNotification) {

	iid := &id.ID{}
	copy(iid[:], identity)

	cb := func(localIdentity *id.ID, deleted bool) {
		pin.Notify(localIdentity[:], deleted)
	}

	c.api.GetStorage().GetEdge().AddUpdateCallback(iid, cb)
}

func (c *Client) GetPreimages(identity []byte) (string, error) {

	iid := &id.ID{}
	copy(iid[:], identity)

	list, exist := c.api.GetStorage().GetEdge().Get(iid)
	if !exist {
		return "", errors.Errorf("Could not find a preimage list for %s", iid)
	}

	marshaled, err := json.Marshal(&list)

	return string(marshaled), err
}
