////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

//go:build !js || !wasm

package sync

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/ekv"
)

// Smoke test of NewCollector.
func TestNewCollector(t *testing.T) {
	baseDir := "TestNewCollector/"
	rng := rand.New(rand.NewSource(42))
	syncPath := baseDir + "collector/"
	txLog := makeTransactionLog(syncPath, password, t)

	// Construct kv
	kv := ekv.MakeMemstore()

	// Create remote kv
	remoteKv, err := newVersionedKV(txLog, kv, nil, nil, nil)
	require.NoError(t, err)

	myId, err := cmix.NewRandomInstanceID(rng)
	require.NoError(t, err)
	cmix.StoreInstanceID(myId, remoteKv)

	workingDir := baseDir + "remoteFsSmoke/"

	fsRemote := NewFileSystemRemoteStorage(workingDir)

	collector := NewCollector(syncPath, txLog, fsRemote, remoteKv)

	expected := &collector{
		syncPath:             syncPath,
		myID:                 myId,
		lastUpdateRead:       make(map[cmix.InstanceID]time.Time, 0),
		synchronizationEpoch: synchronizationEpoch,
		deviceTxTracker:      newDeviceTransactionTracker(),
		txLog:                txLog,
		remote:               fsRemote,
		kv:                   remoteKv,
		myLogPath:            collector.myLogPath,
	}

	require.Equal(t, expected, collector)

	// Delete the test file at the end
	os.RemoveAll(baseDir)

}

func TestNewCollector_CollectChanges(t *testing.T) {
	baseDir := "TestNewCollector_CollectChanges/"

	// Note: these are pre-canned serialized mutate logs w/ transactions
	// with timestamp values in various years (6 timestamps per tx log)
	var remoteTxLogsEnc = []string{
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0ZAAAAAAAAAFhYREtUWExPR0RWQ09GRlNUYm5Wc2JBPT0GAAAAAAAAAJIAAAAAAAAAMCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWWRial9DNnlFc3FuTUk4LXVSNUJlVFBOejZVSXZiSVV5c1FtTTNlbXVuSmN3OWVJYktpeVNwN2pWYWYxdTZlZWcxQWF0WkhxS0FvTnJ6aWRYaUtsZmluU0FsRWhDX3hSZVNqT3dVMGk3R0NadWp1YWWSAAAAAAAAADEsR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txS3l3dExpOHdDRUdqWjdKSG1JY3d4MG9oZ2xwVzhOUWhmZzdUWll1SlBaUnR3alFPaXl1cTlZaFRlR3lEaVFsRHJ3a20tbnZXZk1YTE1sNm1WTmRyZGhPdV8zMWh3cXhJYlRERlBHQW1Kd25hS01UZ29kM2FfV1NqkgAAAAAAAAAyLE1USXpORFUyTnpnNU9qczhQVDRfUUVGQ1EwUkZSa2RJSElDQUlFQ29kbXlvVTdsbGJhaFkxS3B2NGRVTWJ3SVZ6VFVDV09Benp2VmRNLVptUm1FcThqVk1FbHpFbWltTzJaSkhLVmdISGk3aFBQSkxWa0xndThPTTBRa0N2Q1dMcXJnX2tNYk9aOE9YNTFUQZIAAAAAAAAAMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z0dZdnNuWVd6X3lDSFV4Z1J0MXVWT1V1bmRxMk1xdm1pWF9PdlBaWHBjbmRabzFHVTBIM0RQeW5LRm9hRFNrN0wzbmF6Y3JiMzk3a05mWTJPYm9qRC1EbkhieWF0dU9JNnVRdnZUdDJSQTlSOGVYWjKSAAAAAAAAADQsWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5YzNSMWRuZDRtQlJLeE5HeXlpQTFzRlMzOUZxRUlxSmRIeXVaQWIwSHNydF9QTzBYZF90RHlfeENiUTZ6Z0hhblljSXU5eWFST0xfUXFjaFhsejRJNkFoTjYwM3pEMVhVTGVTMkRDUmVvd1dURUFRNG02alBObHQzkgAAAAAAAAA1LGVYcDdmSDEtZjRDQmdvT0VoWWFIaUltS2k0eU5qby1RN1Myb2pvQ2QtRWRHUE55d1Utd2pzUlNLTzV1V0lmZmNsTDhaaGFLOHk0WldsdEtWbFVtMU9QUjhiYkFXdXNlRFdZWVJUS3ZmSkZXRzRYYTNFWDFYZFJLQ1Zva2d5SGx6RGJGVnFpZ2xTbTFZN29fbg==",
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0ZAAAAAAAAAFhYREtUWExPR0RWQ09GRlNUYm5Wc2JBPT0GAAAAAAAAAJIAAAAAAAAAMCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWWRial9DNnlFc3FuTUk4LXVSNUJlVFBONTZVSXZiSVV5c1FtTTNlbXVuSmN3OWVJYktpeVNwN2pWYWYxdTZlZWcxQWF0WkhxS0FvTnJ6aWRYaUtsZmluU0FsRWpaNWtaSnE2OUVCWUUwZlN2NU9wWW2SAAAAAAAAADEsR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txS3l3dExpOHdDRUdqWjdKSG1JY3d4MG9oZ2xwVzhOY2xmZzdUWll1SlBaUnR3alFPaXl1cTlZaFRlR3lEaVFsRHJ3a20tbnZXZk1YTE1sNm1WTmRyZGhPdV8zMWh3cXhJYlRDc1hrcHFva3lUYjVQUmZtUTBWaHdikgAAAAAAAAAyLE1USXpORFUyTnpnNU9qczhQVDRfUUVGQ1EwUkZSa2RJSElDQUlFQ29kbXlvVTdsbGJhaFkxS2xyNGRVTWJ3SVZ6VFVDV09Benp2VmRNLVptUm1FcThqVk1FbHpFbWk2TzJaSkhLVmdISGk3aFBQSkxWa0xqamNPTTBRbUpobXpSZmJfc184RU9VRVF2ajZ0R5IAAAAAAAAAMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z0dZdnNuWVd6X3lDSFV4Z1J0MXVWT1VpamRxMk1xdm1pWF9PdlBaWHBjbmRabzFHVTBIM0RQeW5LRm9hRFNrbkwzbmF6Y3JiMzk3a05mWTJPYm9qQXpqbkhieWJjbEdYbUE3Yy14QlhvR1I1Z3huamqSAAAAAAAAADQsWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5YzNSMWRuZDRtQlJLeE5HeXlpQTFzRlMzOUZxRUlxRlpIeXVaQWIwSHNydF9QTzBYZF90RHlfeENiUTZ6Z0hhblljSXU5eUNST0xfUXFjaFhsejRJNkFoTjYwM3dLVlhVTGVRYU0tU2Y5S2VqeHNUTHdKS0oxbG1kkgAAAAAAAAA1LGVYcDdmSDEtZjRDQmdvT0VoWWFIaUltS2k0eU5qby1RN1Myb2pvQ2QtRWRHUE55d1Utd2pzUmVHTzV1V0lmZmNsTDhaaGFLOHk0WldsdEtWbFVtMU9QUjhiYkFXdXNHRFdZWVJUS3ZmSkZXRzRYYTNFWDFVVXhLQ1ZvbEpxMWFjYUlpbVJwckxOTklLQXBjeQ==",
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0ZAAAAAAAAAFhYREtUWExPR0RWQ09GRlNUYm5Wc2JBPT0GAAAAAAAAAJIAAAAAAAAAMCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWWRial9DNnlFc3FuTUk4LXVSNUJlVFBGdzZVSXZiSVV5c1FtTTNlbXVuSmN3OWVJYktpeVNwN2pWYWYxdTZlZWcxQWF0WkhxS0FvTnJ6aWRYaUtsZmluU0FsRWpZLW9JZG04T3Frb095Wk1xWmZlR2iSAAAAAAAAADEsR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txS3l3dExpOHdDRUdqWjdKSG1JY3d4MG9oZ2xwVzhOWWtmZzdUWll1SlBaUnR3alFPaXl1cTlZaFRlR3lEaVFsRHJ3a20tbmpXZk1YTE1sNm1WTmRyZGhPdV8zMWg5S3hJYlRDM2lZQXlKV2hDbzB4dkpYVmk1bTh4kgAAAAAAAAAyLE1USXpORFUyTnpnNU9qczhQVDRfUUVGQ1EwUkZSa2RJSElDQUlFQ29kbXlvVTdsbGJhaFkxS2hvNGRVTWJ3SVZ6VFVDV09Benp2VmRNLVptUm1FcThqVk1FbHpFbWlxTzJaSkhLVmdISGk3aFBQSkxWa0xnamNPTTBRblBlZElnU18yTEVKUlFYNmVSMzFqcJIAAAAAAAAAMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z0dZdnNuWVd6X3lDSFV4Z1J0MXVWT1VtaWRxMk1xdm1pWF9PdlBaWHBjbmRabzFHVTBIM0RQeW5LRm9hRFNrakwzbmF6Y3JiMzk3a05mWTJPYm9qQTNqbkhieWFCT0N2WWdKYnBka3ZpczE3bzFQRGmSAAAAAAAAADQsWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5YzNSMWRuZDRtQlJLeE5HeXlpQTFzRlMzOUZxRUlxQmVIeXVaQWIwSHNydF9QTzBYZF90RHlfeENiUTZ6Z0hhblljSXU5eWFST0xfUXFjaFhsejRJNkFoTjYwM3pEMVhVTGVSZWZSRUZPV2sxS3ZjVU9tMnZkR01UkgAAAAAAAAA1LGVYcDdmSDEtZjRDQmdvT0VoWWFIaUltS2k0eU5qby1RN1Myb2pvQ2QtRWRHUE55d1Utd2pzUmFITzV1V0lmZmNsTDhaaGFLOHk0WldsdEtWbFVtMU9QUjhiYkFXdXNHRFdZWVJUS3ZmSkZXRzRYYTNFWDFVVXhLQ1ZvbUhXWlI4V3kyT05zMWV0bDVXcTlNaA==",
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0ZAAAAAAAAAFhYREtUWExPR0RWQ09GRlNUYm5Wc2JBPT0GAAAAAAAAAJIAAAAAAAAAMCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWWRial9DNnlFc3FuTUk4LXVSNUJlVFA5dzZVSXZiSVV5c1FtTTNlbXVuSmN3OWVJYktpeVNwN2pWYWYxdTZlZWcxQWF0WkhxS0FvTnJ6aWRYaUtsZmluU0FsRWhKYjFLcUVoNUpiSjVIY2JtQkZ4NHmSAAAAAAAAADEsR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txS3l3dExpOHdDRUdqWjdKSG1JY3d4MG9oZ2xwVzhOZ2tmZzdUWll1SlBaUnR3alFPaXl1cTlZaFRlR3lEaVFsRHJ3a20tbnZXZk1YTE1sNm1WTmRyZGhPdV8zMWh3cXhJYlRDd3diUDc4aU5Hc0dwb0JPdHhyaHRjkgAAAAAAAAAyLE1USXpORFUyTnpnNU9qczhQVDRfUUVGQ1EwUkZSa2RJSElDQUlFQ29kbXlvVTdsbGJhaFkxS1pvNGRVTWJ3SVZ6VFVDV09Benp2VmRNLVptUm1FcThqVk1FbHpFbWltTzJaSkhLVmdISGk3aFBQSkxWa0xndThPTTBRa1Y0aml6Qm1DVDlsTFl4VWV4LUVBTZIAAAAAAAAAMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z0dZdnNuWVd6X3lDSFV4Z1J0MXVWT1VlaWRxMk1xdm1pWF9PdlBaWHBjbmRabzFHVTBIM0RQeW5LRm9hRFNrbkwzbmF6Y3JiMzk3a05mWTJPYm9qQXpqbkhieVl2dkFPdHVxMTJKRTNUd2ZfNkJMcHGSAAAAAAAAADQsWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5YzNSMWRuZDRtQlJLeE5HeXlpQTFzRlMzOUZxRUlxNWZIeXVaQWIwSHNydF9QTzBYZF90RHlfeENiUTZ6Z0hhblljSXU5eUdST0xfUXFjaFhsejRJNkFoTjYwM3dPVlhVTGVRUktqaldSOHhtSW5DbUhEakJzd3EykgAAAAAAAAA1LGVYcDdmSDEtZjRDQmdvT0VoWWFIaUltS2k0eU5qby1RN1Myb2pvQ2QtRWRHUE55d1Utd2pzUmlFTzV1V0lmZmNsTDhaaGFLOHk0WldsdEtWbFVtMU9QUjhiYkFXdXNlRFdZWVJUS3ZmSkZXRzRYYTNFWDFYZFJLQ1Zva09XU2tVSk9YSXl4M19rSXVUU2ZEMg==",
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0ZAAAAAAAAAFhYREtUWExPR0RWQ09GRlNUYm5Wc2JBPT0GAAAAAAAAAJIAAAAAAAAAMCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWWRial9DNnlFc3FuTUk4LXVSNUJmVGZadzZVSXZiSVV5c1FtTTNlbXVuSmN3OWVJYktpeVNwN2pWYWYxdTZlZWcxQWF0WkhxS0FvTnJ6aWRYaUtsZmluU0FsRWlrcDRENVVCQl9jaG5NajJ4bTQ3YzKSAAAAAAAAADEsR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txS3l3dExpOHdDRUdqWjdKSG1JY3d4MG9oZ2xwWDhkRW5mZzdUWll1SlBaUnR3alFPaXl1cTlZaFRlR3lEaVFsRHJ3a20tbnZXZk1YTE1sNm1WTmRyZGhPdV8zMWh3cXhJYlREaEN1aFItRm5lSFVzYWxCUVAtd1JGkgAAAAAAAAAyLE1USXpORFUyTnpnNU9qczhQVDRfUUVGQ1EwUkZSa2RJSElDQUlFQ29kbXlvVTdsbGJhaFoxYTlwNGRVTWJ3SVZ6VFVDV09Benp2VmRNLVptUm1FcThqVk1FbHpFbWlpTzJaSkhLVmdISGk3aFBQSkxWa0xncThPTTBRbk5DNUNlTmVfTzlJM2RlaU9LMUhmUZIAAAAAAAAAMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z0dZdnNuWVd6X3lDSFV4Z1J0MXVVT0U2bGRxMk1xdm1pWF9PdlBaWHBjbmRabzFHVTBIM0RQeW5LRm9hRFNrX0wzbmF6Y3JiMzk3a05mWTJPYm9qRDZEbkhieVp3SENDNzhwR1lLTUtqcVRaR3RtUVOSAAAAAAAAADQsWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5YzNSMWRuZDRtQlJLeE5HeXlpQTFzRlMzOUZxRkk2ZGZIeXVaQWIwSHNydF9QTzBYZF90RHlfeENiUTZ6Z0hhblljSXU5eWFST0xfUXFjaFhsejRJNkFoTjYwM3pEMVhVTGVUQlhwMnZPeEJNSTloNTdEblVZQVN5kgAAAAAAAAA1LGVYcDdmSDEtZjRDQmdvT0VoWWFIaUltS2k0eU5qby1RN1Myb2pvQ2QtRWRHUE55d1Utd2lzQkdMTzV1V0lmZmNsTDhaaGFLOHk0WldsdEtWbFVtMU9QUjhiYkFXdXNlRFdZWVJUS3ZmSkZXRzRYYTNFWDFYZFJLQ1ZvbkIyYUx6MUVoQjllRHp6OWRBd2VLUQ==",
	}

	syncPath := baseDir + "collector/"
	txLog := makeTransactionLog(syncPath, password, t)

	// Construct kv
	kv := ekv.MakeMemstore()

	// Create remote kv
	remoteKv, err := newVersionedKV(txLog, kv, nil, nil, nil)
	require.NoError(t, err)

	workingDir := baseDir + "remoteFsSmoke/"

	fsRemote := NewFileSystemRemoteStorage(workingDir)
	devices := make([]string, 0)
	rng := rand.New(rand.NewSource(42))

	// Construct collector
	myId, _ := cmix.NewRandomInstanceID(rng)
	cmix.StoreInstanceID(myId, remoteKv)
	collector := NewCollector(syncPath, txLog, fsRemote, remoteKv)

	// Write mock data to file (collectChanges will Read from file)
	for _, remoteTxLogEnc := range remoteTxLogsEnc {
		mockInstanceID, err := cmix.NewRandomInstanceID(rng)
		txLogPath := filepath.Join(syncPath,
			fmt.Sprintf(txLogPathFmt, mockInstanceID,
				keyID(txLog.deviceSecret)))

		require.NoError(t, err)
		mockTxLog, err := base64.StdEncoding.DecodeString(remoteTxLogEnc)
		require.NoError(t, err)
		require.NoError(t, fsRemote.Write(txLogPath, mockTxLog))
		devices = append(devices, mockInstanceID.String())
	}

	_, err = collector.collectChanges(devices)
	require.NoError(t, err)

	// Ensure device tracker has proper length for all devices
	for _, dvcIdStr := range devices {
		dvcId, err := cmix.NewInstanceIDFromString(dvcIdStr)
		require.NoError(t, err)
		received := collector.deviceTxTracker.changes[dvcId]
		require.Len(t, received, 6)
	}
	// Delete the test file at the end
	os.RemoveAll(baseDir)
}

func TestCollector_ApplyChanges(t *testing.T) {
	baseDir := "TestCollector_ApplyChanges/"

	// Note: these are pre-canned serialized mutate logs w/ transactions
	// with timestamp values in various years (6 timestamps per tx log)
	var remoteTxLogsEnc = []string{
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0ZAAAAAAAAAFhYREtUWExPR0RWQ09GRlNUYm5Wc2JBPT0GAAAAAAAAAJIAAAAAAAAAMCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWWRial9DNnlFc3FuTUk4LXVSNUJlVFBOejZVSXZiSVV5c1FtTTNlbXVuSmN3OWVJYktpeVNwN2pWYWYxdTZlZWcxQWF0WkhxS0FvTnJ6aWRYaUtsZmluU0FsRWhDX3hSZVNqT3dVMGk3R0NadWp1YWWSAAAAAAAAADEsR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txS3l3dExpOHdDRUdqWjdKSG1JY3d4MG9oZ2xwVzhOUWhmZzdUWll1SlBaUnR3alFPaXl1cTlZaFRlR3lEaVFsRHJ3a20tbnZXZk1YTE1sNm1WTmRyZGhPdV8zMWh3cXhJYlRERlBHQW1Kd25hS01UZ29kM2FfV1NqkgAAAAAAAAAyLE1USXpORFUyTnpnNU9qczhQVDRfUUVGQ1EwUkZSa2RJSElDQUlFQ29kbXlvVTdsbGJhaFkxS3B2NGRVTWJ3SVZ6VFVDV09Benp2VmRNLVptUm1FcThqVk1FbHpFbWltTzJaSkhLVmdISGk3aFBQSkxWa0xndThPTTBRa0N2Q1dMcXJnX2tNYk9aOE9YNTFUQZIAAAAAAAAAMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z0dZdnNuWVd6X3lDSFV4Z1J0MXVWT1V1bmRxMk1xdm1pWF9PdlBaWHBjbmRabzFHVTBIM0RQeW5LRm9hRFNrN0wzbmF6Y3JiMzk3a05mWTJPYm9qRC1EbkhieWF0dU9JNnVRdnZUdDJSQTlSOGVYWjKSAAAAAAAAADQsWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5YzNSMWRuZDRtQlJLeE5HeXlpQTFzRlMzOUZxRUlxSmRIeXVaQWIwSHNydF9QTzBYZF90RHlfeENiUTZ6Z0hhblljSXU5eWFST0xfUXFjaFhsejRJNkFoTjYwM3pEMVhVTGVTMkRDUmVvd1dURUFRNG02alBObHQzkgAAAAAAAAA1LGVYcDdmSDEtZjRDQmdvT0VoWWFIaUltS2k0eU5qby1RN1Myb2pvQ2QtRWRHUE55d1Utd2pzUlNLTzV1V0lmZmNsTDhaaGFLOHk0WldsdEtWbFVtMU9QUjhiYkFXdXNlRFdZWVJUS3ZmSkZXRzRYYTNFWDFYZFJLQ1Zva2d5SGx6RGJGVnFpZ2xTbTFZN29fbg==",
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0ZAAAAAAAAAFhYREtUWExPR0RWQ09GRlNUYm5Wc2JBPT0GAAAAAAAAAJIAAAAAAAAAMCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWWRial9DNnlFc3FuTUk4LXVSNUJlVFBONTZVSXZiSVV5c1FtTTNlbXVuSmN3OWVJYktpeVNwN2pWYWYxdTZlZWcxQWF0WkhxS0FvTnJ6aWRYaUtsZmluU0FsRWpaNWtaSnE2OUVCWUUwZlN2NU9wWW2SAAAAAAAAADEsR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txS3l3dExpOHdDRUdqWjdKSG1JY3d4MG9oZ2xwVzhOY2xmZzdUWll1SlBaUnR3alFPaXl1cTlZaFRlR3lEaVFsRHJ3a20tbnZXZk1YTE1sNm1WTmRyZGhPdV8zMWh3cXhJYlRDc1hrcHFva3lUYjVQUmZtUTBWaHdikgAAAAAAAAAyLE1USXpORFUyTnpnNU9qczhQVDRfUUVGQ1EwUkZSa2RJSElDQUlFQ29kbXlvVTdsbGJhaFkxS2xyNGRVTWJ3SVZ6VFVDV09Benp2VmRNLVptUm1FcThqVk1FbHpFbWk2TzJaSkhLVmdISGk3aFBQSkxWa0xqamNPTTBRbUpobXpSZmJfc184RU9VRVF2ajZ0R5IAAAAAAAAAMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z0dZdnNuWVd6X3lDSFV4Z1J0MXVWT1VpamRxMk1xdm1pWF9PdlBaWHBjbmRabzFHVTBIM0RQeW5LRm9hRFNrbkwzbmF6Y3JiMzk3a05mWTJPYm9qQXpqbkhieWJjbEdYbUE3Yy14QlhvR1I1Z3huamqSAAAAAAAAADQsWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5YzNSMWRuZDRtQlJLeE5HeXlpQTFzRlMzOUZxRUlxRlpIeXVaQWIwSHNydF9QTzBYZF90RHlfeENiUTZ6Z0hhblljSXU5eUNST0xfUXFjaFhsejRJNkFoTjYwM3dLVlhVTGVRYU0tU2Y5S2VqeHNUTHdKS0oxbG1kkgAAAAAAAAA1LGVYcDdmSDEtZjRDQmdvT0VoWWFIaUltS2k0eU5qby1RN1Myb2pvQ2QtRWRHUE55d1Utd2pzUmVHTzV1V0lmZmNsTDhaaGFLOHk0WldsdEtWbFVtMU9QUjhiYkFXdXNHRFdZWVJUS3ZmSkZXRzRYYTNFWDFVVXhLQ1ZvbEpxMWFjYUlpbVJwckxOTklLQXBjeQ==",
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0ZAAAAAAAAAFhYREtUWExPR0RWQ09GRlNUYm5Wc2JBPT0GAAAAAAAAAJIAAAAAAAAAMCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWWRial9DNnlFc3FuTUk4LXVSNUJlVFBGdzZVSXZiSVV5c1FtTTNlbXVuSmN3OWVJYktpeVNwN2pWYWYxdTZlZWcxQWF0WkhxS0FvTnJ6aWRYaUtsZmluU0FsRWpZLW9JZG04T3Frb095Wk1xWmZlR2iSAAAAAAAAADEsR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txS3l3dExpOHdDRUdqWjdKSG1JY3d4MG9oZ2xwVzhOWWtmZzdUWll1SlBaUnR3alFPaXl1cTlZaFRlR3lEaVFsRHJ3a20tbmpXZk1YTE1sNm1WTmRyZGhPdV8zMWg5S3hJYlRDM2lZQXlKV2hDbzB4dkpYVmk1bTh4kgAAAAAAAAAyLE1USXpORFUyTnpnNU9qczhQVDRfUUVGQ1EwUkZSa2RJSElDQUlFQ29kbXlvVTdsbGJhaFkxS2hvNGRVTWJ3SVZ6VFVDV09Benp2VmRNLVptUm1FcThqVk1FbHpFbWlxTzJaSkhLVmdISGk3aFBQSkxWa0xnamNPTTBRblBlZElnU18yTEVKUlFYNmVSMzFqcJIAAAAAAAAAMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z0dZdnNuWVd6X3lDSFV4Z1J0MXVWT1VtaWRxMk1xdm1pWF9PdlBaWHBjbmRabzFHVTBIM0RQeW5LRm9hRFNrakwzbmF6Y3JiMzk3a05mWTJPYm9qQTNqbkhieWFCT0N2WWdKYnBka3ZpczE3bzFQRGmSAAAAAAAAADQsWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5YzNSMWRuZDRtQlJLeE5HeXlpQTFzRlMzOUZxRUlxQmVIeXVaQWIwSHNydF9QTzBYZF90RHlfeENiUTZ6Z0hhblljSXU5eWFST0xfUXFjaFhsejRJNkFoTjYwM3pEMVhVTGVSZWZSRUZPV2sxS3ZjVU9tMnZkR01UkgAAAAAAAAA1LGVYcDdmSDEtZjRDQmdvT0VoWWFIaUltS2k0eU5qby1RN1Myb2pvQ2QtRWRHUE55d1Utd2pzUmFITzV1V0lmZmNsTDhaaGFLOHk0WldsdEtWbFVtMU9QUjhiYkFXdXNHRFdZWVJUS3ZmSkZXRzRYYTNFWDFVVXhLQ1ZvbUhXWlI4V3kyT05zMWV0bDVXcTlNaA==",
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0ZAAAAAAAAAFhYREtUWExPR0RWQ09GRlNUYm5Wc2JBPT0GAAAAAAAAAJIAAAAAAAAAMCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWWRial9DNnlFc3FuTUk4LXVSNUJlVFA5dzZVSXZiSVV5c1FtTTNlbXVuSmN3OWVJYktpeVNwN2pWYWYxdTZlZWcxQWF0WkhxS0FvTnJ6aWRYaUtsZmluU0FsRWhKYjFLcUVoNUpiSjVIY2JtQkZ4NHmSAAAAAAAAADEsR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txS3l3dExpOHdDRUdqWjdKSG1JY3d4MG9oZ2xwVzhOZ2tmZzdUWll1SlBaUnR3alFPaXl1cTlZaFRlR3lEaVFsRHJ3a20tbnZXZk1YTE1sNm1WTmRyZGhPdV8zMWh3cXhJYlRDd3diUDc4aU5Hc0dwb0JPdHhyaHRjkgAAAAAAAAAyLE1USXpORFUyTnpnNU9qczhQVDRfUUVGQ1EwUkZSa2RJSElDQUlFQ29kbXlvVTdsbGJhaFkxS1pvNGRVTWJ3SVZ6VFVDV09Benp2VmRNLVptUm1FcThqVk1FbHpFbWltTzJaSkhLVmdISGk3aFBQSkxWa0xndThPTTBRa1Y0aml6Qm1DVDlsTFl4VWV4LUVBTZIAAAAAAAAAMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z0dZdnNuWVd6X3lDSFV4Z1J0MXVWT1VlaWRxMk1xdm1pWF9PdlBaWHBjbmRabzFHVTBIM0RQeW5LRm9hRFNrbkwzbmF6Y3JiMzk3a05mWTJPYm9qQXpqbkhieVl2dkFPdHVxMTJKRTNUd2ZfNkJMcHGSAAAAAAAAADQsWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5YzNSMWRuZDRtQlJLeE5HeXlpQTFzRlMzOUZxRUlxNWZIeXVaQWIwSHNydF9QTzBYZF90RHlfeENiUTZ6Z0hhblljSXU5eUdST0xfUXFjaFhsejRJNkFoTjYwM3dPVlhVTGVRUktqaldSOHhtSW5DbUhEakJzd3EykgAAAAAAAAA1LGVYcDdmSDEtZjRDQmdvT0VoWWFIaUltS2k0eU5qby1RN1Myb2pvQ2QtRWRHUE55d1Utd2pzUmlFTzV1V0lmZmNsTDhaaGFLOHk0WldsdEtWbFVtMU9QUjhiYkFXdXNlRFdZWVJUS3ZmSkZXRzRYYTNFWDFYZFJLQ1Zva09XU2tVSk9YSXl4M19rSXVUU2ZEMg==",
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0ZAAAAAAAAAFhYREtUWExPR0RWQ09GRlNUYm5Wc2JBPT0GAAAAAAAAAJIAAAAAAAAAMCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWWRial9DNnlFc3FuTUk4LXVSNUJmVGZadzZVSXZiSVV5c1FtTTNlbXVuSmN3OWVJYktpeVNwN2pWYWYxdTZlZWcxQWF0WkhxS0FvTnJ6aWRYaUtsZmluU0FsRWlrcDRENVVCQl9jaG5NajJ4bTQ3YzKSAAAAAAAAADEsR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txS3l3dExpOHdDRUdqWjdKSG1JY3d4MG9oZ2xwWDhkRW5mZzdUWll1SlBaUnR3alFPaXl1cTlZaFRlR3lEaVFsRHJ3a20tbnZXZk1YTE1sNm1WTmRyZGhPdV8zMWh3cXhJYlREaEN1aFItRm5lSFVzYWxCUVAtd1JGkgAAAAAAAAAyLE1USXpORFUyTnpnNU9qczhQVDRfUUVGQ1EwUkZSa2RJSElDQUlFQ29kbXlvVTdsbGJhaFoxYTlwNGRVTWJ3SVZ6VFVDV09Benp2VmRNLVptUm1FcThqVk1FbHpFbWlpTzJaSkhLVmdISGk3aFBQSkxWa0xncThPTTBRbk5DNUNlTmVfTzlJM2RlaU9LMUhmUZIAAAAAAAAAMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z0dZdnNuWVd6X3lDSFV4Z1J0MXVVT0U2bGRxMk1xdm1pWF9PdlBaWHBjbmRabzFHVTBIM0RQeW5LRm9hRFNrX0wzbmF6Y3JiMzk3a05mWTJPYm9qRDZEbkhieVp3SENDNzhwR1lLTUtqcVRaR3RtUVOSAAAAAAAAADQsWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5YzNSMWRuZDRtQlJLeE5HeXlpQTFzRlMzOUZxRkk2ZGZIeXVaQWIwSHNydF9QTzBYZF90RHlfeENiUTZ6Z0hhblljSXU5eWFST0xfUXFjaFhsejRJNkFoTjYwM3pEMVhVTGVUQlhwMnZPeEJNSTloNTdEblVZQVN5kgAAAAAAAAA1LGVYcDdmSDEtZjRDQmdvT0VoWWFIaUltS2k0eU5qby1RN1Myb2pvQ2QtRWRHUE55d1Utd2lzQkdMTzV1V0lmZmNsTDhaaGFLOHk0WldsdEtWbFVtMU9QUjhiYkFXdXNlRFdZWVJUS3ZmSkZXRzRYYTNFWDFYZFJLQ1ZvbkIyYUx6MUVoQjllRHp6OWRBd2VLUQ==",
	}

	syncPath := baseDir + "collector/"
	txLog := makeTransactionLog(syncPath, password, t)

	// Construct kv
	kv := ekv.MakeMemstore()

	// Create remote kv
	remoteKv, err := newVersionedKV(txLog, kv, nil, nil, nil)
	require.NoError(t, err)

	workingDir := baseDir + "remoteFsSmoke/"
	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	// Write mock data to file (collectChanges will Read from file)
	fsRemote := NewFileSystemRemoteStorage(workingDir)
	devices := make([]string, 0)
	for i, remoteTxLogEnc := range remoteTxLogsEnc {
		mockInstanceID := strconv.Itoa(i)
		mockTxLog, err := base64.StdEncoding.DecodeString(remoteTxLogEnc)
		require.NoError(t, err)
		require.NoError(t, fsRemote.Write(mockInstanceID, mockTxLog))
		devices = append(devices, mockInstanceID)
	}

	// Construct collector
	rng := rand.New(rand.NewSource(42))
	myId, _ := cmix.NewRandomInstanceID(rng)
	cmix.StoreInstanceID(myId, remoteKv)
	collector := NewCollector(syncPath, txLog, fsRemote, remoteKv)
	_, err = collector.collectChanges(devices)
	require.NoError(t, err)
	require.NoError(t, collector.applyChanges())

}

// Unit test for devicePatchTracker.AddToDevice.
func TestDeviceTransactionTracker_AddToDevice(t *testing.T) {
	const numTests = 100
	dvcTracker := newDeviceTransactionTracker()

	// Construct transactions
	changes := make([]Mutate, 0)
	for i := 0; i < numTests; i++ {
		iStr := strconv.Itoa(i)
		key, val := "key"+iStr, "val"+iStr
		offset := time.Now().Add(time.Duration(i) * time.Second)
		tx := NewMutate(offset, key, []byte(val))
		changes = append(changes, tx)
	}

	// Add changes to tracker
	rng := rand.New(rand.NewSource(42))
	instanceID, _ := cmix.NewRandomInstanceID(rng)
	dvcTracker.AddToDevice(instanceID, changes)

	// Ensure changes have been put into tracker
	require.Equal(t, changes, dvcTracker.changes[instanceID])
}

// Unit test for devicePatchTracker.Sort.
func TestDeviceTransactionTracker_Next(t *testing.T) {
	const numTests = 100
	dvcTracker := newDeviceTransactionTracker()

	// Construct transactions
	changes := make([]Mutate, 0)
	for i := 0; i < numTests; i++ {
		iStr := strconv.Itoa(i)
		key, val := "key"+iStr, "val"+iStr
		offset := time.Duration(i) * time.Minute
		if i%2 == 0 {
			offset = offset * -1
		}
		offsetTs := time.Now().Add(offset)
		tx := NewMutate(offsetTs, key, []byte(val))
		changes = append(changes, tx)
	}

	// Add changes to tracker
	rng := rand.New(rand.NewSource(42))
	instanceID, _ := cmix.NewRandomInstanceID(rng)
	dvcTracker.AddToDevice(instanceID, changes)

	// Ensure next retrieves changes put into tracker
	ordered := dvcTracker.Sort()

	// Ensure ordered is indeed sorted
	require.True(t, sort.SliceIsSorted(ordered, func(i, j int) bool {
		first, second := ordered[i].Timestamp, ordered[j].Timestamp
		return first.Before(second)
	}))

	// Manually sort changes
	sort.Slice(changes, func(i, j int) bool {
		first, second := changes[i].Timestamp, changes[j].Timestamp
		return first.Before(second)
	})

	// Ensure sorted changes matches
	require.Equal(t, changes, ordered)

	// Construct new transactions
	newChanges := make([]Mutate, 0)
	for i := 0; i < numTests; i++ {
		iStr := strconv.Itoa(i)
		key, val := "keyAfterNext"+iStr, "valAfterNext"+iStr

		offsetTs := time.Now().Add(time.Duration(i))
		tx := NewMutate(offsetTs, key, []byte(val))
		newChanges = append(newChanges, tx)
	}

	// Add new transactions to tracker
	dvcTracker.AddToDevice(instanceID, newChanges)

	// Ensure next retrieves the new mutate list
	require.Equal(t, newChanges, dvcTracker.Sort())
}
