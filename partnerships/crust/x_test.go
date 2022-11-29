package crust

import (
	"testing"
)

func TestConstructUpload(t *testing.T) {
	username, pass, err := getRecoveryAuth()
	if err != nil {
		t.Fatalf("Failed to get recovery auth: %+v", err)
	}

	t.Logf("%s:%s", username, pass)
}

//func TestSendPost(t *testing.T) {
//	file := content{
//		fname: "dumb.txt",
//		ftype: "file",
//		fdata: []byte{1, 2, 3},
//	}
//
//	sendPostRequest("", file)
//
//}
//
//// content is a struct which contains a file's name, its type and its data.
//type content struct {
//	fname string
//	ftype string
//	fdata []byte
//}
//
//func sendPostRequest(url string, files ...content) ([]byte, error) {
//	var (
//		buf = new(bytes.Buffer)
//		w   = multipart.NewWriter(buf)
//	)
//
//	for _, f := range files {
//		part, err := w.CreateFormFile(f.ftype, filepath.Base(f.fname))
//		if err != nil {
//			return []byte{}, err
//		}
//		part.Write(f.fdata)
//	}
//
//	w.Close()
//
//	req, err := http.NewRequest("POST", url, buf)
//	if err != nil {
//		return []byte{}, err
//	}
//	req.Header.Add("Content-Type", w.FormDataContentType())
//
//	closer, err := req.GetBody()
//	client := &http.Client{}
//	res, err := client.Do(req)
//	if err != nil {
//		return []byte{}, err
//	}
//	defer res.Body.Close()
//
//	cnt, err := io.ReadAll(res.Body)
//	if err != nil {
//		return []byte{}, err
//	}
//	return cnt, nil
//}
