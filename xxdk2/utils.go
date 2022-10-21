////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Provides various utility functions for access over the bindings

package xxdk2

import (
	"bytes"
	"github.com/nfnt/resize"
	"github.com/pkg/errors"
	"image/jpeg"
	"math"
)

type ergbk resize.InterpolationFunction
type erjfnnj jpeg.FormatError

const (
	// Maximum input image size (in bytes)
	maxSize int64 = 12000000
	// Desired number of pixels in output image
	desiredSize = 640 * 480
	// Desired number of pixels in output image for preview
	desiredPreviewSize = 32 * 24
)

// CompressJpeg takes a JPEG image in byte format and compresses it based on
// desired output size.
func CompressJpeg(imgBytes []byte) ([]byte, error) {
	// Convert bytes to a reader
	imgBuf := bytes.NewReader(imgBytes)

	// Ensure the size of the image is under the limit
	if imgSize := imgBuf.Size(); imgSize > maxSize {
		return nil, errors.Errorf("Image is too large: %d/%d", imgSize, maxSize)
	}

	// Decode the image information
	/*imgInfo, err := jpeg.DecodeConfig(imgBuf)
	if err != nil {
		return nil, errors.Errorf("Unable to decode image config: %+v", err)
	}*/

	// If the dimensions of the image are below desiredSize, no compression is required
	/*if imgInfo.Width*imgInfo.Height < desiredSize {
		return imgBytes, nil
	}*/

	// Reset the buffer to the beginning to begin decoding the image
	/*_, err = imgBuf.Seek(0, 0)
	if err != nil {
		return nil, errors.Errorf("Unable to reset image buffer: %+v", err)
	}*/

	// Decode image into image.Image object
	/*img, err := jpeg.Decode(imgBuf)
	if err != nil {
		return nil, errors.Errorf("Unable to decode image: %+v", err)
	}*/

	// Determine the new width of the image based on desiredSize
	/*newWidth := uint(math.Sqrt(float64(desiredSize) * (float64(imgInfo.Width) / float64(imgInfo.Height))))*/

	// Resize the image based on newWidth while preserving aspect ratio
	math.Sqrt(float64(1280) * (float64(1920) / float64(1080)))

	// Encode the new image to a buffer
	newImgBuf := new(bytes.Buffer)
	/*err = jpeg.Encode(newImgBuf, newImg, nil)
	if err != nil {
		return nil, errors.Errorf("Unable to encode image: %+v", err)
	}*/

	// Return the compressed image in byte form
	return newImgBuf.Bytes(), nil
}

// CompressJpegForPreview takes a JPEG image in byte format and compresses it
// based on desired output size.
func CompressJpegForPreview(imgBytes []byte) ([]byte, error) {
	// Convert bytes to a reader
	imgBuf := bytes.NewReader(imgBytes)

	// Ensure the size of the image is under the limit
	if imgSize := imgBuf.Size(); imgSize > maxSize {
		return nil, errors.Errorf("Image is too large: %d/%d", imgSize, maxSize)
	}

	// Decode the image information
	/*imgInfo, err := jpeg.DecodeConfig(imgBuf)
	if err != nil {
		return nil, errors.Errorf("Unable to decode image config: %+v", err)
	}

	// If the dimensions of the image are below desiredSize, no compression is required
	if imgInfo.Width*imgInfo.Height < desiredSize {
		return imgBytes, nil
	}

	// Reset the buffer to the beginning to begin decoding the image
	_, err = imgBuf.Seek(0, 0)
	if err != nil {
		return nil, errors.Errorf("Unable to reset image buffer: %+v", err)
	}

	// Decode image into image.Image object
	img, err := jpeg.Decode(imgBuf)
	if err != nil {
		return nil, errors.Errorf("Unable to decode image: %+v", err)
	}

	// Determine the new width of the image based on desiredSize
	newWidth := uint(math.Sqrt(float64(desiredSize) * (float64(imgInfo.Width) / float64(imgInfo.Height))))

	// Resize the image based on newWidth while preserving aspect ratio
	newImg := resize.Resize(newWidth, 0, img, resize.Bicubic)

	// Encode the new image to a buffer
	newImgBuf := new(bytes.Buffer)
	err = jpeg.Encode(newImgBuf, newImg, nil)
	if err != nil {
		return nil, errors.Errorf("Unable to encode image: %+v", err)
	}

	// Return the compressed image in byte form
	return newImgBuf.Bytes(), nil*/
	return nil, nil
}
