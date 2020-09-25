package fact

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
)

type Fact struct {
	Fact string
	T    Type
}

func (f Fact) Get() string {
	return f.Fact
}

func (f Fact) GetType() int {
	return int(f.T)
}

func (f Fact) Copy() interfaces.Fact {
	f2 := Fact{
		Fact: f.Fact,
		T:    f.T,
	}
	return &f2
}

// marshal is for transmission for UDB, not a part of the fact interface
func (f Fact) Marshal() []byte {
	serial := []byte(f.Fact)
	b := make([]byte, len(serial)+1)
	b[0] = byte(f.T)

	copy(b[1:len(serial)-1], serial)
	return b
}

func Unmarshal(b []byte) (Fact, error) {
	t := Type(b[0])
	if !t.IsValid() {
		return Fact{}, errors.Errorf("Fact is not a valid type: %s", t)
	}

	f := string(b[1:])

	return Fact{
		Fact: f,
		T:    t,
	}, nil
}
