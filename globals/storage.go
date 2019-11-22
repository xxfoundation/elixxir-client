////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"os"
	"sync"
)

const (
	NoSave uint8 = iota
	LocationA
	LocationB
)

type Storage interface {
	SetLocation(string, string) error
	GetLocation() (string, string)
	SaveA([]byte) error
	SaveB([]byte) error
	LoadA() []byte
	LoadB() []byte
	IsEmpty() bool
}

type DefaultStorage struct {
	locationA string
	locationB string
	sync.Mutex
}

func (ds *DefaultStorage) SetLocation(locationA, locationB string) error {
	ds.Lock()
	ds.locationA = locationA
	ds.locationB = locationB
	ds.Unlock()
	return nil
}

func (ds *DefaultStorage) GetLocation() (string, string) {
	ds.Lock()
	defer ds.Unlock()
	return ds.locationA, ds.locationB
}

func (ds *DefaultStorage) IsEmpty() bool {
	_, err := os.Stat(ds.locationA)
	firstEmpty := err != nil && !os.IsNotExist(err)
	_, err = os.Stat(ds.locationB)
	secondEmpty := err != nil && !os.IsNotExist(err)
	return firstEmpty && secondEmpty
}

func (ds *DefaultStorage) SaveA(data []byte) error {
	return dsSaveHelper(ds.locationA, data)
}

func (ds *DefaultStorage) LoadA() []byte {
	return dsLoadHelper(ds.locationA)
}



func (ds *DefaultStorage) SaveB(data []byte) error {
	return dsSaveHelper(ds.locationB, data)
}

func (ds *DefaultStorage) LoadB() []byte {
	return dsLoadHelper(ds.locationB)
}


type RamStorage struct {
	DataA []byte
	DataB []byte
}

func (rs *RamStorage) SetLocation(string, string) error {
	return nil
}

func (rs *RamStorage) GetLocation() (string, string) {
	return "", ""
}

func (rs *RamStorage) SaveA(data []byte) error {
	rs.DataA = make([]byte, len(data))
	copy(rs.DataA, data)
	return nil
}

func (rs *RamStorage) SaveB(data []byte) error {
	rs.DataB = make([]byte, len(data))
	copy(rs.DataB, data)
	return nil
}

func (rs *RamStorage) LoadA() []byte {
	b := make([]byte, len(rs.DataA))
	copy(b, rs.DataA)

	return b
}

func (rs *RamStorage) LoadB() []byte {
	b := make([]byte, len(rs.DataB))
	copy(b, rs.DataB)

	return b
}

func (rs *RamStorage) IsEmpty() bool {
	return (rs.DataA == nil || len(rs.DataA) == 0) && (rs.DataB == nil || len(rs.DataB) == 0)
}

func dsLoadHelper(loc string) []byte {
	// Check if the file exists, return nil if it does not
	finfo, err1 := os.Stat(loc)

	if err1 != nil {
		Log.ERROR.Printf("Default Storage Load: Unknown Error Occurred on"+
			" file check: \n  %v", err1.Error())
		return nil
	}

	b := make([]byte, finfo.Size())

	// Open the file, return nil if it cannot be opened
	f, err2 := os.Open(loc)

	defer func() {
		if f != nil {
			f.Close()
		} else {
			Log.WARN.Println("Could not close file, file is nil")
		}
	}()

	if err2 != nil {
		Log.ERROR.Printf("Default Storage Load: Unknown Error Occurred on"+
			" file open: \n  %v", err2.Error())
		return nil
	}

	// Read the data from the file, return nil if read fails
	_, err3 := f.Read(b)

	if err3 != nil {
		Log.ERROR.Printf("Default Storage Load: Unknown Error Occurred on"+
			" file read: \n  %v", err3.Error())
		return nil
	}

	return b

}

func dsSaveHelper(loc string, data []byte) error {
	//check if the file exists, delete if it does
	_, err1 := os.Stat(loc)

	if err1 == nil {
		errRmv := os.Remove(loc)
		if errRmv != nil {
			Log.WARN.Printf("Could not remove Storage File B: %s", errRmv)
		}
	} else if !os.IsNotExist(err1) {
		Log.ERROR.Printf("Default Storage Save: Unknown Error Occurred on"+
			" file check: \n  %v",
			err1.Error())
		return err1
	}

	//create new file
	f, err2 := os.Create(loc)

	defer func() {
		if f != nil {
			f.Close()
		} else {
			Log.WARN.Println("Could not close file, file is nil")
		}
	}()

	if err2 != nil {
		Log.ERROR.Printf("Default Storage Save: Unknown Error Occurred on"+
			" file creation: \n %v", err2.Error())
		return err2
	}

	//Save to file
	_, err3 := f.Write(data)

	if err3 != nil {
		Log.ERROR.Printf("Default Storage Save: Unknown Error Occurred on"+
			" file write: \n %v", err3.Error())
		return err3
	}

	return nil
}
