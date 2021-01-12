package ud

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	jww "github.com/spf13/jwalterweatherman"
)

type Manager struct {
	//external
	client *api.Client
	comms   *client.Comms
	rng     *fastRNG.StreamGenerator
	sw      interfaces.Switchboard
	storage *storage.Session
	net interfaces.NetworkManager

	//loaded from external access
	udID *id.ID
	privKey *rsa.PrivateKey
	grp *cyclic.Group

	//internal maps
	host    *connect.Host
	inProgressLookup    map[uint64]chan *LookupResponse
	inProgressLookupMux sync.RWMutex

	inProgressSearch    map[uint64]chan *SearchResponse
	inProgressSearchMux sync.RWMutex

	//State tracking
	commID     uint64
	commIDLock sync.Mutex

	registered *uint32
}

// New manager builds a new user discovery manager. It requires that an
// updated NDF is available and will error if one is not.
func NewManager(client *api.Client)(*Manager, error){
	jww.INFO.Println("ud.NewManager()")
	if !client.GetHealth().IsHealthy(){
		return nil, errors.New("cannot start UD Manager when network " +
			"is not healthy")
	}

	m := &Manager{
		client:				 client,
		comms:               client.GetComms(),
		rng:                 client.GetRng(),
		sw:                  client.GetSwitchboard(),
		storage:             client.GetStorage(),
		net:                 client.GetNetworkInterface(),
		inProgressLookup: 	 make(map[uint64]chan *LookupResponse),
		inProgressSearch:	 make(map[uint64]chan *SearchResponse),
	}

	var err error

	//check that user discovery is available in the ndf
	def := m.net.GetInstance().GetPartialNdf().Get()
	if m.udID, err = id.Unmarshal(def.UDB.ID); err!=nil{
		return nil, errors.WithMessage(err,"NDF does not have User " +
			"Discovery information, is there network access?: ID could not be " +
			"unmarshaled")
	}

	if def.UDB.Cert==""{
		return nil, errors.New("NDF does not have User " +
			"Discovery information, is there network access?: Cert " +
			"not present")
	}

	//create the user discovery host object

	hp := connect.GetDefaultHostParams()
	if m.host, err = m.comms.AddHost(m.udID, def.UDB.Address,[]byte(def.UDB.Cert),
		hp); err!=nil{
		return nil, errors.WithMessage(err, "User Discovery host " +
			"object could not be constructed")
	}

	//get the commonly used data from storage
	m.privKey = m.storage.GetUser().RSA

	//load the last used commID
	m.loadCommID()

	//load if the client is registered
	m.loadRegistered()

	//store the pointer to the group locally for easy access
	m.grp = m.storage.E2e().GetGroup()

	return m, nil
}

func (m *Manager) StartProcesses()  {
	m.client.AddService(m.startProcesses)
}

func (m *Manager) startProcesses() stoppable.Stoppable {

	lookupStop := stoppable.NewSingle("UDLookup")
	lookupChan := make(chan message.Receive, 100)
	m.sw.RegisterChannel("UDLookupResponse", m.udID, message.UdLookupResponse, lookupChan)
	go m.lookupProcess(lookupChan, lookupStop.Quit())

	searchStop := stoppable.NewSingle("UDSearch")
	searchChan := make(chan message.Receive, 100)
	m.sw.RegisterChannel("UDSearchResponse", m.udID, message.UdSearchResponse, searchChan)
	go m.searchProcess(searchChan, searchStop.Quit())

	udMulti := stoppable.NewMulti("UD")
	udMulti.Add(lookupStop)
	udMulti.Add(searchStop)
	return lookupStop
}




