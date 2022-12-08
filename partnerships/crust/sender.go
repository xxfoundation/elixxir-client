////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package crust

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
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
	defer resp.Body.Close()
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	jww.INFO.Printf("[CRUST] Response data: %v", string(responseData))
	jww.INFO.Printf("[Crust] Response status code: %d", resp.StatusCode)
	// Handle error code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, errors.Errorf("received error code %d (details: %v)",
			resp.StatusCode, handleError(responseData))
	}

	return responseData, nil
}

// errorResponse handles any POST or GET request returning an error as a
// response.
type errorResponse struct {
	Error struct {
		Reason  string `json:"reason"`
		Details string `json:"details"`
	} `json:"error"`
}

// handleError converts the response data which contains a JSON encoded error
// string to the error context that is standard in Golang.
func handleError(responseData []byte) error {
	errResponse := &errorResponse{}
	err := json.Unmarshal(responseData, errResponse)
	if err != nil {
		return err
	}

	errResp := fmt.Sprintf("%s: %s",
		errResponse.Error.Reason, errResponse.Error.Details)

	return errors.New(errResp)
}
