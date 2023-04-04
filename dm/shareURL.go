////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"encoding/base64"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"io"
	goUrl "net/url"
	"strconv"
)

// The current version number of the share URL structure.
const shareUrlVersion = 1

const (
	// Channel.ShareURL
	parseHostUrlErr = "could not parse host URL: %+v"

	// DecodeShareURL
	parseShareUrlErr   = "could not parse URL: %+v"
	urlVersionErr      = "no version found"
	parseVersionErr    = "failed to parse version: %+v"
	versionErr         = "version mismatch: require v%d, found v%d"
	decodePublicUrlErr = "could not decode public share URL: %+v"
)

const (
	versionKey = "v"
	myTokenKey = "t"
	myEcPubKey = "p"

	// MaxUsesKey is the key used to save max uses in a URL. The value is
	// expected to be a positive integer.
	MaxUsesKey = "m"
)

// ShareURL generates a URL that can be used to share this channel with others
// on the given host.
func ShareURL(url string, maxUses int, token int, key nike.PublicKey,
	csprng io.Reader) (string, error) {
	u, err := goUrl.Parse(url)
	if err != nil {
		return "", errors.Errorf(parseHostUrlErr, err)
	}

	q := u.Query()
	q.Set(versionKey, strconv.Itoa(shareUrlVersion))
	q.Set(MaxUsesKey, strconv.Itoa(maxUses))

	u.RawQuery = encodePublicShareURL(q, token, key).Encode()

	u.RawQuery = q.Encode()

	return u.String(), nil

}

// DecodeShareURL decodes the given URL for information to DM another user.
func DecodeShareURL(url string, password string) (int, nike.PublicKey, error) {
	u, err := goUrl.Parse(url)
	if err != nil {
		return 0, nil, errors.Errorf(parseShareUrlErr, err)
	}

	q := u.Query()

	// Check the version
	versionString := q.Get(versionKey)
	if versionString == "" {
		return 0, nil, errors.New(urlVersionErr)
	}
	v, err := strconv.Atoi(versionString)
	if err != nil {
		return 0, nil, errors.Errorf(parseVersionErr, err)
	} else if v != shareUrlVersion {
		return 0, nil, errors.Errorf(versionErr, shareUrlVersion, v)
	}

	// Decode the URL based on the information available (e.g., only the public
	// URL has a salt, so if the saltKey is specified, it is a public URL)
	partnerToken, partnerPublicKey, err := decodePublicShareURL(q)
	if err != nil {
		return 0, nil, errors.Errorf(decodePublicUrlErr, err)

	}

	return partnerToken, partnerPublicKey, nil
}

// encodePublicShareURL encodes the channel to a Public share URL.
func encodePublicShareURL(q goUrl.Values, token int, key nike.PublicKey) goUrl.Values {
	q.Set(myTokenKey, strconv.FormatUint(uint64(token), 10))
	q.Set(myEcPubKey, base64.URLEncoding.EncodeToString(key.Bytes()))
	return q
}

// decodePublicShareURL decodes the values in the url.Values from a public DM
// URL to the data encoded within (including the DM token and [nike.PublicKey]).
func decodePublicShareURL(q goUrl.Values) (int, nike.PublicKey, error) {
	// Retrieve the token
	dmToken, err := strconv.Atoi(q.Get(myTokenKey))
	if err != nil {
		return 0, nil, err
	}

	// Retrieve the key data
	rsaKeyData, err := base64.URLEncoding.DecodeString(q.Get(myEcPubKey))
	if err != nil {
		return 0, nil, err
	}

	// Unmarshal the public key
	pubKey := ecdh.ECDHNIKE.NewEmptyPublicKey()
	err = pubKey.FromBytes(rsaKeyData)
	if err != nil {
		return 0, nil, err
	}

	return dmToken, pubKey, nil
}
