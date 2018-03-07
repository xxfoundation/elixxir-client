package globals

import (
"os"
jww "github.com/spf13/jwalterweatherman"
)


var LocalStorage Storage

func InitStorage(store Storage, location string)(bool){
	if LocalStorage!=nil {
		jww.ERROR.Printf("Invalid Local Storage Creation: Local storage" +
			" already created")
		return false
	}

	var intermediateStorage Storage

	if store == nil{
		intermediateStorage = &DefaultStorage{}
	}else{
		intermediateStorage = store
	}

	success := intermediateStorage.SetLocation(location)

	if !success{
		jww.ERROR.Printf("Invalid Local Storage Location")
		return false
	}

	LocalStorage = intermediateStorage

	return true
}

type Storage interface{
	SetLocation(string)(bool)
	Save([]byte)(bool)
	Load()([]byte)
}

type DefaultStorage struct{
	location string
}

func (ds DefaultStorage) SetLocation (location string) (bool){
	ds.location = location
	return true
}


func (ds DefaultStorage) Save(data []byte) bool{
	//check if the file exists, delete if it does
	_, err1 := os.Stat(ds.location)

	if err1 == nil {
		os.Remove(ds.location)
	}else if !os.IsNotExist(err1){
		jww.ERROR.Printf("Default Storage Save: Unknown Error Occurred on" +
			" file check: \n  %v",
			err1.Error())
		return false
	}

	//create new file
	f, err2 := os.Create(ds.location)

	defer f.Close()

	if err2!=nil {
		jww.ERROR.Printf("Default Storage Save: Unknown Error Occurred on" +
			" file creation: \n %v",	err2.Error())
		return false
	}

	//Save to file
	_, err3 := f.Write(data)

	if err3 != nil{
		jww.ERROR.Printf("Default Storage Save: Unknown Error Occurred on" +
			" file write: \n %v",	err3.Error())
		return false
	}

	return true
}

func (ds DefaultStorage)Load() ([]byte){
	// Check if the file exists, return nil if it does not
	finfo, err1 := os.Stat(ds.location)

	if err1 != nil {
		jww.ERROR.Printf("Default Storage Load: Unknown Error Occurred on" +
			" file check: \n  %v", 	err1.Error())
		return nil
	}

	b := make([]byte, finfo.Size())

	// Open the file, return nil if it cannot be opened
	f, err2 := os.Open(ds.location)

	defer f.Close()

	if err2 != nil {
		jww.ERROR.Printf("Default Storage Load: Unknown Error Occurred on" +
			" file open: \n  %v", 	err2.Error())
		return nil
	}

	// Read the data from the file, return nil if read fails
	_, err3 := f.Read(b)

	if err3 != nil {
		jww.ERROR.Printf("Default Storage Load: Unknown Error Occurred on" +
			" file read: \n  %v", 	err3.Error())
		return nil
	}

	return b

}

type RamStorage struct{
	data []byte
}

func (rs RamStorage) SetLocation (location string) (bool){
	return true
}

func (rs RamStorage) Save(data []byte)(bool){
	rs.data = make([]byte,len(data))
	copy(rs.data,data)
	return true
}

func (rs RamStorage) Load() ([]byte){
	b := make([]byte,len(rs.data))
	copy(b,rs.data)
	return b
}

