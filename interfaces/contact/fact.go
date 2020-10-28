package contact

type Fact struct {
	Fact string
	T    FactType
}

func NewFact(ft FactType, fact string) (Fact, error) {
	//todo: filter the fact string
	return Fact{
		Fact: fact,
		T:    ft,
	}, nil
}

// marshal is for transmission for UDB, not a part of the fact interface
func (f Fact) Stringify() string {
	return f.T.Stringify() + f.Fact
}

func UnstringifyFact(s string) (Fact, error) {
	ft, err := UnstringifyFactType(s)
	if err != nil {
		return Fact{}, err
	}

	return NewFact(ft, s)
}
