////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package distributedBackup

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
)

// sendRequest is a helper function which handles the sending of a request
// and any potential error within the response.
func sendRequest(req *http.Request) ([]byte, error) {
	// Send POST request
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	// Read response
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Handle error
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Could not upload backup: %v",
			handleError(responseData)))
	}

	return responseData, nil
}

// errorResponse handles any POST or GET request returning an error as a
// response.
type errorResponse struct {
	Error string
}

// handleError converts the response data which contains a JSON encoded error
// string to the error context that is standard in Golang.
func handleError(responseData []byte) error {
	errResponse := &errorResponse{}
	err := json.Unmarshal(responseData, errResponse)
	if err != nil {
		return err
	}

	return errors.New(errResponse.Error)
}
