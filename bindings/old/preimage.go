package old

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"unsafe"
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

func (c *Client) GetPreimages(identity []byte) string {
	iid := &id.ID{}
	copy(iid[:], identity)

	list, exist := c.api.GetStorage().GetEdge().Get(iid)
	if !exist {
		jww.ERROR.Printf("Preimage for %s does not exist", iid.String())
		return ""
	}

	marshaled, err := json.Marshal(&list)
	if err != nil {
		jww.ERROR.Printf("Error marshaling preimages: %s", err.Error())
		return ""
	}

	jww.DEBUG.Printf("Preimages size: %v %v %d",
		reflect.TypeOf(marshaled).Align(), unsafe.Sizeof(marshaled), len(marshaled))
	return string(marshaled)
}
