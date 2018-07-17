package payment

type MessageHash [512]byte

/*
type TransactionList struct {
	transactionMap *map[MessageHash]Transaction
	mutex          sync.Mutex
	value          uint64
}

func NewTransactionList(tag string) (*TransactionList, error) {
	gob.Register(TransactionList{})

	var tlmPtr *map[MessageHash]Transaction

	tli, err := user.TheSession.QueryMap(tag)

	if err != nil {
		//If there is an err make the object
		tlMap := make(map[MessageHash]Transaction)
		tlmPtr = &tlMap

		if err == user.ErrQuery {
			err = user.TheSession.UpsertMap(tag, tlmPtr)
		}
		if err != nil {
			return nil, err
		}
	} else {
		tlmPtr = tli.(*map[MessageHash]Transaction)
	}

	value := uint64(0)

	for _, t := range *tlmPtr {
		value += t.Value
	}

	return &TransactionList{transactionMap: tlmPtr, value: value}, nil
}

func (tl *TransactionList) Value() uint64 {
	tl.mutex.Lock()
	v := tl.value
	tl.mutex.Unlock()
	return v
}

func (tl *TransactionList) add(mh MessageHash, t Transaction) (error) {
	if len(*tl.transactionMap) == 0 {
		*tl.transactionMap[]
	} else {
		for i := 0; i < len(*os.list); i++ {
			if (*os.list)[i].Value() > cs.Value() {
				tmp := append((*os.list)[:i], cs)
				*os.list = append(tmp, (*os.list)[i:]...)
			}
		}
	}

	os.value += cs.Value()
}

func (os *OrderedStorage) Add(cs coin.Sleeve) {
	os.mutex.Lock()
	os.add(cs)
	os.mutex.Unlock()
}

func (os *OrderedStorage) pop(index uint64) coin.Sleeve {
	if uint64(len(*os.list)) >= index {
		return coin.Sleeve{}
	}

	cs := (*os.list)[index]

	*os.list = append((*os.list)[:index], (*os.list)[index+1:]...)

	os.value -= cs.Value()

	return cs
}

func (os *OrderedStorage) Pop(index uint64) coin.Sleeve {
	os.mutex.Lock()
	cs := os.Pop(index)
	os.mutex.Unlock()
	return cs
}

func (os *OrderedStorage) get(index uint64) coin.Sleeve {
	if uint64(len(*os.list)) >= index {
		return coin.Sleeve{}
	}

	return (*os.list)[index]
}

func (os *OrderedStorage) Get(index uint64) coin.Sleeve {
	os.mutex.Lock()
	cs := os.get(index)
	os.mutex.Unlock()
	return cs
}
*/