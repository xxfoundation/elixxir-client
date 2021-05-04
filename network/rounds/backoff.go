package rounds
//
//import (
//	"container/list"
//	"github.com/golang/protobuf/ptypes/timestamp"
//	"gitlab.com/elixxir/client/storage/versioned"
//	"gitlab.com/xx_network/primitives/id"
//	"sync"
//	"time"
//)
//
//type UncheckableRounds struct {
//	store *Storeage
//	backoffChan chan *rounds
//	mux sync.Mutex
//
//}
//
//func NewUncheckableRounds(kv *versioned.KV, backoff chan *rounds) (*UncheckableRounds, error) {
//	vo, err :=  kv.Get("AllUnchecked", version)
//	if err != nil {
//		// handle error
//	}
//
//	roundList := deserialize(vo.Data)
//
//	return &UncheckableRounds{list: roundList, backoffChan: backoff}, nil
//}
//
//func (ur *UncheckableRounds) Add(rid id.Round)  {
//	ur.mux.Lock()
//	defer ur.mux.Unlock()
//	data, exists := ur.list[rid]
//	if exists {
//		// handle increment
//		data.numTries++
//		data.ts = time.Now()
//	}
//
//	ur.backoffChan <- data
//}
//
//func (ur *UncheckableRounds) Remove(rid id.Round)  {
//	ur.mux.Lock()
//	defer ur.mux.Unlock()
//	delete(ur.list, rid)
//	// delete from store
//}
//
//// scheduler method vs event based method
//// iterate finding rounds due for a check, send them along
//// to processRetrieval. Tick for params amount
//// For high and low channel sending, event based is better
////
//
//func (m *Manager) processRoundBackoff(comms messageRetrievalComms,
//	quitCh <-chan struct{}) {
//
//	done := false
//	for !done {
//		select {
//		case <-quitCh:
//			done = true
//
//		case rnd := <- m.backoffChan:
//			// generate version object
//			timeSleep := isRoundCheckDue(rnd.numTries)
//			go m.sendBackoff(timeSleep)
//		}
//	}
//}
//
//
//func (m *Manager) sendBackoff(ts time.Duration, round *rounds)  {
//	go func() {
//
//		time.Sleep(ts)
//
//		m.lookupRoundMessages <- round
//
//
//		data := serialize(rounds{
//			rid: rid,
//			numTries: round.numTries++,
//			ts: time.now(),
//		})
//
//		newVO := versioned.Object{
//			// fill in other fields
//			Data:      data,
//		}
//
//	}()
//
//}
//
//
//
//func (s *Storeage) Notatate(rid id.Round, backoffChan chan rounds)  {
//
//	// Check if it exists, modify that object
//	vo, err := s.kv.Get(prefix+rid, version)
//	if err != nil {
//		// handle
//		return
//	}
//
//	// Returns new round if nil passed in (round doesn't exist in memory
//	round := deserializeRoundCheck(vo.Data)
//
//	backoffChan <- round
//
//
//
//}
//
