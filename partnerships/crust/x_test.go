package crust

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"strings"
	"testing"
)

func TestConstructUpload(t *testing.T) {

	var client *http.Client
	var remoteURL string
	{
		//setup a mocked http client.
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, err := httputil.DumpRequest(r, true)
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s", b)
		}))
		defer ts.Close()
		client = ts.Client()
		remoteURL = ts.URL
	}

	//prepare the reader instances to encode
	values := map[string]io.Reader{
		"file":  mustOpen("/home/josh/src/clientMaster/partnerships/crust/file.go"), // lets assume its this file
		"other": strings.NewReader("hello world!"),
	}
	err := Upload(client, remoteURL, values)
	if err != nil {
		panic(err)
	}
}

func Upload(client *http.Client, url string, values map[string]io.Reader) (err error) {
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for key, r := range values {
		var fw io.Writer
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}
		// Add an image file
		if x, ok := r.(*os.File); ok {
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				return
			}
		} else {
			// Add other fields
			if fw, err = w.CreateFormField(key); err != nil {
				return
			}
		}
		if _, err = io.Copy(fw, r); err != nil {
			return err
		}

	}
	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	bodyContents, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic("Failed to read body")
	}

	fmt.Printf("%s\n", string(bodyContents))

	return
}

func mustOpen(f string) *os.File {
	r, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	return r
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
