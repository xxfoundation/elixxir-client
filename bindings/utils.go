////////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Provides various utility functions for access over the bindings

package bindings

import "gitlab.com/elixxir/client/api"

// CompressJpeg takes a JPEG image in byte format
// and compresses it based on desired output size
func CompressJpeg(imgBytes []byte) ([]byte, error) {
	return api.CompressJpeg(imgBytes)
}
