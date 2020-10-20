package bind

type Contact interface {
	GetID() []byte
	GetDHPublicKey() []byte
	GetOwnershipProof() []byte
	GetFactList() FactList
	Marshal() ([]byte, error)
}

type FactList interface {
	Num() int
	Get(int) Fact
	Add(string, int) error
}

type Fact interface {
	Get() string
	Type() int
}
