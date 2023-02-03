package sync

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"gitlab.com/elixxir/crypto/hash"
	"io"
	"strconv"
	"sync"
)

const (
	xxdkTxLogHeader = "XXDKTXLOGHDR"
)

type TransactionLog struct {
	remote RemoteStore
	path   string // the current path on remote and local storage

	local LocalStore // possibly object.versioned or other. Make it an interface.

	hdr    Header
	txs    []Transaction    // This list must always be ordered
	curBuf bufio.ReadWriter // need to look up a write buffer

	deviceSecret []byte
	lck          sync.RWMutex

	rng io.Reader
}

func (tl *TransactionLog) Append(t Transaction) error {

	tl.lck.Lock()
	defer tl.lck.Unlock()

	headerMarshal, err := json.Marshal(tl.hdr)
	if err != nil {
		// todo: better err
		return err
	}

	_, err = tl.curBuf.WriteString(xxdkTxLogHeader +
		base64.URLEncoding.EncodeToString(headerMarshal))
	if err != nil {
		// todo: better err
		return err
	}

	for i := 0; i < len(tl.txs); i++ {

		h, err := hash.NewCMixHash()
		if err != nil {
			// todo: better err, possibly panic
			return err
		}

		h.Write(binary.LittleEndian.AppendUint16(make([]byte, 0), uint16(i)))
		h.Write(tl.deviceSecret)
		secret := h.Sum(nil)

		txMarshal, err := json.Marshal(tl.txs[i])
		if err != nil {
			// todo: better err
			return err
		}

		encrypted := encrypt(txMarshal, string(secret), tl.rng)

		_, err = tl.curBuf.WriteString(strconv.Itoa(i) + "," +
			base64.URLEncoding.EncodeToString(encrypted))
		if err != nil {
			// todo: better err
			return err
		}

	}

	// when we finish reading..

	return tl.Save()
}

func (tl *TransactionLog) Save() error {
	dataToSave := make([]byte, 0)
	_, err := io.ReadFull(tl.curBuf, dataToSave)
	if err != nil {
		// todo: better err
		return err
	}

	err = tl.curBuf.Flush()
	if err != nil {
		// todo: better err
		return err
	}

	err = tl.local.Write(tl.path, dataToSave)
	if err != nil {
		// todo: better err
		return err
	}

	return tl.remote.Write(tl.path, dataToSave)
}
