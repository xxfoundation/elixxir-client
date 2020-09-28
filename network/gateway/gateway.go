package gateway

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"io"
	"math"
)

type HostGetter interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
}

func Get(ndf *ndf.NetworkDefinition, hg HostGetter, rng io.Reader) (*connect.Host, error) {
	// Get a random gateway
	gateways := ndf.Gateways
	gwIdx := ReadRangeUint32(0, uint32(len(gateways)), rng)
	gwID, err := id.Unmarshal(ndf.Nodes[gwIdx].ID)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get Gateway")
	}
	gwID.SetType(id.Gateway)
	gwHost, ok := hg.GetHost(gwID)
	if !ok {
		return nil, errors.Errorf("host for gateway %s could not be "+
			"retrieved", gwID)
	}
	return gwHost, nil
}

func GetLast(hg HostGetter, ri *mixmessages.RoundInfo) (*connect.Host, error) {
	roundTop := ri.GetTopology()
	lastGw, err := id.Unmarshal(roundTop[len(roundTop)-1])
	if err != nil {
		return nil, err
	}
	lastGw.SetType(id.Gateway)

	gwHost, ok := hg.GetHost(lastGw)
	if !ok {
		return nil, errors.Errorf("Could not find host for gateway %s", lastGw)
	}
	return gwHost, nil
}

// ReadUint32 reads an integer from an io.Reader (which should be a CSPRNG)
func ReadUint32(rng io.Reader) uint32 {
	var rndBytes [4]byte
	i, err := rng.Read(rndBytes[:])
	if i != 4 || err != nil {
		panic(fmt.Sprintf("cannot read from rng: %+v", err))
	}
	return binary.BigEndian.Uint32(rndBytes[:])
}

// ReadRangeUint32 reduces an integer from 0, MaxUint32 to the range start, end
func ReadRangeUint32(start, end uint32, rng io.Reader) uint32 {
	size := end - start
	// note we could just do the part inside the () here, but then extra
	// can == size which means a little bit of range is wastes, either
	// choice seems negligible so we went with the "more correct"
	extra := (math.MaxUint32%size + 1) % size
	limit := math.MaxUint32 - extra
	// Loop until we read something inside the limit
	for {
		res := ReadUint32(rng)
		if res > limit {
			continue
		}
		return (res % size) + start
	}
}
