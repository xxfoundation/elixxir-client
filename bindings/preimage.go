package bindings

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/edge"
	"gitlab.com/xx_network/primitives/id"
)


type PreimageNotification interface{
	Notify(identity []byte, deleted bool)
}

func (c *Client)RegisterPreimageCallback(identity []byte, pin PreimageNotification){

	iid := &id.ID{}
	copy(iid[:], identity)

	cb := func(localIdentity *id.ID, deleted bool){
		pin.Notify(localIdentity[:],deleted)
	}

	c.api.GetStorage().GetEdge().AddUpdateCallback(iid, cb)
}

func (c *Client)GetPreimages(identity []byte)(*PreimageList, error){

	iid := &id.ID{}
	copy(iid[:], identity)

	list, exist := c.api.GetStorage().GetEdge().Get(iid)
	if !exist{
		return nil, errors.Errorf("Could not find a preimage list for %s", iid)
	}

	return &PreimageList{list: list}, nil
}

type Preimage struct{
	pi edge.Preimage
}

func (pi *Preimage)Get()[]byte{
	return pi.pi.Data
}

func (pi *Preimage)Type()string{
	return pi.pi.Type
}

func (pi *Preimage)Source()[]byte{
	return pi.pi.Source
}


type PreimageList struct{
	list edge.Preimages
}

func (pil *PreimageList)Len()int{
	return len(pil.list)
}

func (pil *PreimageList)Get(index int)*Preimage{
	return &Preimage{pi:pil.list[index]}
}