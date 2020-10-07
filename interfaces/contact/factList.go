package contact

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
)

type FactList struct {
	source *Contact
}

func (fl FactList) Num() int {
	return len(fl.source.Facts)
}

func (fl FactList) Get(i int) interfaces.Fact {
	return fl.source.Facts[i]
}

func (fl FactList) Add(fact string, factType int) error {
	ft := FactType(factType)
	if !ft.IsValid() {
		return errors.New("Invalid fact type")
	}
	fl.source.Facts = append(fl.source.Facts, Fact{
		Fact: fact,
		T:    ft,
	})
	return nil
}
