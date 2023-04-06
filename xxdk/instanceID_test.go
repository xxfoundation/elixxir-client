////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/ekv"
)

// TestInstanceID performs basic smoke testing of the type
func TestInstanceID(t *testing.T) {
	rng := rand.New(rand.NewSource(8675309))
	generated, err := generateInstanceID(rng)

	require.NoError(t, err)
	require.Equal(t, instanceIDLength, len(generated[:]))

	gStr := generated.String()

	kv := ekv.MakeMemstore()
	err = StoreInstanceID(generated, kv)
	require.NoError(t, err)

	loaded, err := LoadInstanceID(kv)
	require.NoError(t, err)
	require.Equal(t, generated, loaded)

	loadedStr := fmt.Sprintf("%s", loaded)
	require.Equal(t, gStr, loadedStr)

	kv.SetBytes(instanceIDKey, []byte("invalid"))
	invalid, err := LoadInstanceID(kv)
	require.Error(t, err)
	require.Equal(t, InstanceID{}, invalid)

	kv.Delete(instanceIDKey)
	invalid2, err := LoadInstanceID(kv)
	require.Error(t, err)
	require.Equal(t, InstanceID{}, invalid2)

	kv.SetBytes(instanceIDKey, []byte(""))
	invalid3, err := LoadInstanceID(kv)
	require.Error(t, err)
	require.Equal(t, InstanceID{}, invalid3)
}
