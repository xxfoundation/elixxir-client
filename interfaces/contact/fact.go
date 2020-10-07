package contact

import (
	"github.com/pkg/errors"
)

type Fact struct {
	Fact string
	T    FactType
}

func (f Fact) Get() string {
	return f.Fact
}

func (f Fact) Type() int {
	return int(f.T)
}

// marshal is for transmission for UDB, not a part of the fact interface
func (f Fact) Marshal() []byte {
	serial := []byte(f.Fact)
	b := make([]byte, len(serial)+1)
	b[0] = byte(f.T)

	copy(b[1:len(serial)-1], serial)
	return b
}

func UnmarshalFact(b []byte) (Fact, error) {
	t := FactType(b[0])
	if !t.IsValid() {
		return Fact{}, errors.Errorf("Fact is not a valid type: %s", t)
	}

	f := string(b[1:])

	return Fact{
		Fact: f,
		T:    t,
	}, nil
}
