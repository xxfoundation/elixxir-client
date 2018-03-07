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

	intermediateStorage, err := intermediateStorage.SetLocation(location)

	if err != nil{
		jww.ERROR.Printf("Invalid Local Storage Location: %s", err.Error())
		return false
	}

	LocalStorage = intermediateStorage

	return true
}

type Storage interface{
	SetLocation(string)(Storage, error)
	GetLocation()(string)
	Save([]byte)(Storage, error)
	Load()([]byte)
}

type DefaultStorage struct{
	location string
}

func (ds DefaultStorage) SetLocation (location string) (Storage, error){
	ds.location = location
	return ds, nil
}

func (ds DefaultStorage) GetLocation () (string) {
	return ds.location
}


func (ds DefaultStorage) Save(data []byte) (Storage, error){
	//check if the file exists, delete if it does
	_, err1 := os.Stat(ds.location)

	if err1 == nil {
		//jww.INFO.Printf("Storage file already exists, deleting.")
		os.Remove(ds.location)
	}else if !os.IsNotExist(err1){
		jww.ERROR.Printf("Default Storage Save: Unknown Error Occurred on" +
			" file check: \n  %v",
			err1.Error())
		return ds, err1
	}

	//create new file
	f, err2 := os.Create(ds.location)

	defer f.Close()

	if err2!=nil {
		jww.ERROR.Printf("Default Storage Save: Unknown Error Occurred on" +
			" file creation: \n %v",	err2.Error())
		return ds, err2
	}

	//Save to file
	_, err3 := f.Write(data)

	if err3 != nil{
		jww.ERROR.Printf("Default Storage Save: Unknown Error Occurred on" +
			" file write: \n %v",	err3.Error())
		return ds, err3
	}

	return ds, nil
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
	data *[]byte
	//data []byte
}

func (rs RamStorage) SetLocation (location string) (Storage, error){
	return rs, nil
}

func (rs RamStorage) GetLocation () (string) {
	return ""
}

func (rs RamStorage) Save(data []byte)(Storage, error){
	*rs.data = data
	return rs, nil
}

func (rs RamStorage) Load() ([]byte){
	return *rs.data
}

