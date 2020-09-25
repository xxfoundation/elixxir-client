package interfaces

type Contact interface {
	GetID() []byte
	GetDHPublicKey() []byte
	AddFact(Fact) Contact
	NumFacts() int
	GetFact(int) (Fact, error)
	Marshal() ([]byte, error)
}

type Fact interface {
	Get() string
	GetType() int
}
