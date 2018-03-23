////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	jww "github.com/spf13/jwalterweatherman"
	"os"
	"errors"
)

var LocalStorage Storage

func InitStorage(store Storage, location string) error {
	if LocalStorage != nil {
		errStr := "Invalid Local Storage Creation: Local storage already created"
		jww.ERROR.Printf(errStr)
		panic(errStr)
	}

	var intermediateStorage Storage

	if store == nil {
		intermediateStorage = &DefaultStorage{}
	} else {
		intermediateStorage = store
	}

	err := intermediateStorage.SetLocation(location)

	if err != nil {
		err = errors.New("Invalid Local Storage Location: " + err.Error())
		jww.ERROR.Printf(err.Error())
		return err
	}

	LocalStorage = intermediateStorage

	return nil
}

type Storage interface {
	SetLocation(string) (error)
	GetLocation() string
	Save([]byte) (error)
	Load() []byte
}

type DefaultStorage struct {
	location string
}

func (ds *DefaultStorage) SetLocation(location string) (error) {
	ds.location = location
	return nil
}

func (ds *DefaultStorage) GetLocation() string {
	return ds.location
}

func (ds *DefaultStorage) Save(data []byte) (error) {
	//check if the file exists, delete if it does
	_, err1 := os.Stat(ds.location)

	if err1 == nil {
		//jww.INFO.Printf("Storage file already exists, deleting.")
		os.Remove(ds.location)
	} else if !os.IsNotExist(err1) {
		jww.ERROR.Printf("Default Storage Save: Unknown Error Occurred on"+
			" file check: \n  %v",
			err1.Error())
		return err1
	}

	//create new file
	f, err2 := os.Create(ds.location)

	defer f.Close()

	if err2 != nil {
		jww.ERROR.Printf("Default Storage Save: Unknown Error Occurred on"+
			" file creation: \n %v", err2.Error())
		return err2
	}

	//Save to file
	_, err3 := f.Write(data)

	if err3 != nil {
		jww.ERROR.Printf("Default Storage Save: Unknown Error Occurred on"+
			" file write: \n %v", err3.Error())
		return err3
	}

	return nil
}

func (ds *DefaultStorage) Load() []byte {
	// Check if the file exists, return nil if it does not
	finfo, err1 := os.Stat(ds.location)

	if err1 != nil {
		jww.ERROR.Printf("Default Storage Load: Unknown Error Occurred on"+
			" file check: \n  %v", err1.Error())
		return nil
	}

	b := make([]byte, finfo.Size())

	// Open the file, return nil if it cannot be opened
	f, err2 := os.Open(ds.location)

	defer f.Close()

	if err2 != nil {
		jww.ERROR.Printf("Default Storage Load: Unknown Error Occurred on"+
			" file open: \n  %v", err2.Error())
		return nil
	}

	// Read the data from the file, return nil if read fails
	_, err3 := f.Read(b)

	if err3 != nil {
		jww.ERROR.Printf("Default Storage Load: Unknown Error Occurred on"+
			" file read: \n  %v", err3.Error())
		return nil
	}

	return b

}

type RamStorage struct {
	data []byte
	//data []byte
}

func (rs *RamStorage) SetLocation(location string) (error) {
	return nil
}

func (rs *RamStorage) GetLocation() string {
	return ""
}

func (rs *RamStorage) Save(data []byte) (error) {
	rs.data = make([]byte, len(data))
	copy(rs.data, data)
	return nil
}

func (rs *RamStorage) Load() []byte {
	b := make([]byte, len(rs.data))
	copy(b, rs.data)

	return b
}
