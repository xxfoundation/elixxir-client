////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"encoding/base64"
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"os"
	"strconv"
	"testing"
	"time"
)

// Smoke test of NewCollector.
func TestNewCollector(t *testing.T) {
	syncPath := baseDir + "collector/"
	txLog := makeTransactionLog(syncPath, password, t)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	remoteKv, err := NewOrLoadRemoteKv(txLog, kv, nil, nil, nil)
	require.NoError(t, err)

	myId := "testingMyId"

	workingDir := baseDir + "remoteFsSmoke/"
	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	fsRemote := NewFileSystemRemoteStorage(workingDir)

	collector := NewCollector(syncPath, myId, txLog, fsRemote, remoteKv)

	expected := &Collector{
		syncPath:             syncPath,
		myID:                 myId,
		lastUpdates:          make(changeLogger, 0),
		SynchronizationEpoch: synchronizationEpoch,
		txLog:                txLog,
		remote:               fsRemote,
		kv:                   remoteKv,
	}

	require.Equal(t, expected, collector)

}

func TestCollector_collectChanges(t *testing.T) {
	// Note: these are pre-canned serialized transaction logs w/ transactions
	// with timestamp values in the year 2053
	var remoteTxLogsEnc = []string{
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0GAAAAAAAAAJIAAAAAAAAAMCwtdnY4X2Y0QUFRSURCQVVHQndnSkNnc01EUTRQRUJFU3h0Z2kydWFXZ2xNeGFLV2JTLTUxNkZyNDNobE5kX2NPVjIzdTczaklYT2I1RldOTUFrOFluTDBqY05fRkZjLTNRd1FkV2FFOW1pcjg4X3dRYmtGQU9XNWYwSmhhZkR5YzZWQmdXUldQYUx5N2Y1YUiSAAAAAAAAADEsRXhRVkZoY1lHUm9iSEIwZUh5QWhJaU1rSlNZbktDa3F3Rkhkc29QZjNQWU50UzRkM0xnejZQaDl6Y3NrZ1pnbG9fOWhTZXFzb3RMWjNYRGlXbzhucmtEblZKSlhleld5RjRDVk5YWlZFX2VZdHJVLUE4RzdtTVcxbkt0d2JST2JCN1dfODhRaVFxX3JTU1o2kgAAAAAAAAAyLEt5d3RMaTh3TVRJek5EVTJOemc1T2pzOFBUNF9RRUZDQzA3dzBTTUo0eTVzQTJkVERCYXEwdV9nc3pMcXJUX0lKNHVDUjZkV2Vhb0tnU2hZTUZ2SHM0ZE5QLWlZSVVER1B1bXZEdFlkbUhwc09pbHVWaWhPTzl6S0VES1FiNVk4WWk5a1hmWlhwdFUyYVgtZ5IAAAAAAAAAMyxRMFJGUmtkSVNVcExURTFPVDFCUlVsTlVWVlpYV0ZsYWpUZU1BMl9KZUxObGpZRFl3MExYeU9idlNmOHR0VXBlSm8tTzNPeUtMR1lhZ1FRd1ZjUF9RR1RoSENxeWk3NHlER3kxNzF1UktCWGdydWgtS29nWDlNa0VTY3lBdWNKRXJhVDEwMUxlQWFDRW1rTHqSAAAAAAAAADQsVzF4ZFhsOWdZV0pqWkdWbVoyaHBhbXRzYlc1dmNIRnlmTTdWTGNyYVUtbjNTMk9LX0hXZmtFTVp0WEZ5UkdNRmJKQ3ZGaFJfSlFJSFNwTFF4UUw3SHNMNklCSXlkMmFYcGFDSWxaU1JLSG8zMkFzSzA3WVlMZEYxM1FTbVpaTWNjUFgtSERkd1dDVm13VmFpkgAAAAAAAAA1LGMzUjFkbmQ0ZVhwN2ZIMS1mNENCZ29PRWhZYUhpSW1LX1g4S3ZhUWtFZnhHR01wSTZLcVM1dDA4SERtTTdzQlllT3NEdDRaODd4Y0x1dkY1MWRtY3ktbVBEdUJpME9nZ0U3cjJnejUteFpBSEtMaU9lMnk4WERKWVhDZm5uNUZwb1Q5WTB3Nk1LLWVsTjQzWA==",
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0GAAAAAAAAAJYAAAAAAAAAMCwtdnY4X2Y0QUFRSURCQVVHQndnSkNnc01EUTRQRUJFU3h0Z2kydWFXZ2xNeGFLV2JTLTUxNkZyNDNobE5kX2NPVjIzdTczaklYT2I1RldOTUFrOFluTDBqY1BqcEk3Q2tUUXBwYnF3a2ltM2s2N291R0Y1Q09pa256OGRYSlRqRjJKeTgwQzhVUFNUR3hZbTZyQT09lgAAAAAAAAAxLEV4UVZGaGNZR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txd0ZIZHNvUGYzUFlOdFM0ZDNMZ3o2UGg5emNza2daZ2xvXzloU2Vxc290TFozWERpV284bnJrRG5WTFY3VFVtaUdZN2hBbnRNQTdDQXJ2TUFkZDY1cllMQmdfU3NJeDR4dXR0RG5PUWd5V1lzQTNpNzdnPT2WAAAAAAAAADIsS3l3dExpOHdNVEl6TkRVMk56ZzVPanM4UFQ0X1FFRkNDMDd3MFNNSjR5NXNBMmRUREJhcTB1X2dzekxxclRfSUo0dUNSNmRXZWFvS2dTaFlNRnZIczRkTlA4LTBGejNYTU9mYk9kc0VpRDEwSW05UUlEZE1IcHU2RDIxaTVQNkgzaG5qRGE3aXptWk56dDExY3c9PZYAAAAAAAAAMyxRMFJGUmtkSVNVcExURTFPVDFCUlVsTlVWVlpYV0ZsYWpUZU1BMl9KZUxObGpZRFl3MExYeU9idlNmOHR0VXBlSm8tTzNPeUtMR1lhZ1FRd1ZjUF9RR1RoSEEyZXZjUWtBbUxCMkZhSU9GTDR0cTVBWEpjVzU0NW9WcE4tWjI1V0J2OHlxUUxrRkx0d2dJM0podz09lgAAAAAAAAA0LFcxeGRYbDlnWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5Zk03VkxjcmFVLW4zUzJPS19IV2ZrRU1adFhGeVJHTUZiSkN2RmhSX0pRSUhTcExReFFMN0hzTDZJRFVlUVIyQXE2NzhvcG1JT0QwdndFMDBwYWtaTHBZZHdsc3lMeEV4SnFpSEVJa21LSVdKdHNZQmZ3PT2WAAAAAAAAADUsYzNSMWRuZDRlWHA3ZkgxLWY0Q0Jnb09FaFlhSGlJbUtfWDhLdmFRa0VmeEdHTXBJNktxUzV0MDhIRG1NN3NCWWVPc0R0NFo4N3hjTHV2RjUxZG1jeS1tUERzZE81cFl5SGJTQ3RETm4xZGNmTVA2d0RYTy1UM1VrUTNqRHBqS0ZvbUFtMTFNOFl3X3QwNTBWU0E9PQ==",
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0GAAAAAAAAAJYAAAAAAAAAMCwtdnY4X2Y0QUFRSURCQVVHQndnSkNnc01EUTRQRUJFU3h0Z2kydWFXZ2xNeGFLV2JTLTUxNkZyNDNobE5kX2NPVjIzdTczaklYT2I1RldOTUFrOFluTDBqY05QUERwejVGaGRwRk84ZW5tU3p0THBIQ21nX1VTVURpS0FYRmFVVTFhLVlsNHEteHp4SnZ1Ujc5SnVVlgAAAAAAAAAxLEV4UVZGaGNZR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txd0ZIZHNvUGYzUFlOdFM0ZDNMZ3o2UGg5emNza2daZ2xvXzloU2Vxc290TFozWERpV284bnJrRG5WSjVkWUdYOFFwRGhlRGgyRjduWDhmTnBaLWpFeG83cHhKX3NoVEVZZVdzNmZHZF9XYlBycnZCcnVpOFmWAAAAAAAAADIsS3l3dExpOHdNVEl6TkRVMk56ZzVPanM4UFQ0X1FFRkNDMDd3MFNNSjR5NXNBMmRUREJhcTB1X2dzekxxclRfSUo0dUNSNmRXZWFvS2dTaFlNRnZIczRkTlAtU1NPaEdJYV9qYlE1Zy1uRFFqZlc4NU1nRXhkWmVXU0FJaVBxaGdYaWtvZTVJV3RBbk9fT1p0bTQzRJYAAAAAAAAAMyxRMFJGUmtkSVNVcExURTFPVDFCUlVsTlVWVlpYV0ZsYWpUZU1BMl9KZUxObGpZRFl3MExYeU9idlNmOHR0VXBlSm8tTzNPeUtMR1lhZ1FRd1ZjUF9RR1RoSENhNGtPaDhXWHJCb2hXeUxGdXY2YTRwVHFGcmpJSllFZUEtRUcwd0VEQlU1NXA3ZkgxMENSRHZ0b1NClgAAAAAAAAA0LFcxeGRYbDlnWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5Zk03VkxjcmFVLW4zUzJPS19IV2ZrRU1adFhGeVJHTUZiSkN2RmhSX0pRSUhTcExReFFMN0hzTDZJQjQ0YkRIWjhMZjgyTnF5TERSNG4wMWR0NTlrUlpvcGhTeHlibjFfZVRFeXNiV1VmcmI5V1ZsQndVYjmWAAAAAAAAADUsYzNSMWRuZDRlWHA3ZkgxLWY0Q0Jnb09FaFlhSGlJbUtfWDhLdmFRa0VmeEdHTXBJNktxUzV0MDhIRG1NN3NCWWVPc0R0NFo4N3hjTHV2RjUxZG1jeS1tUER1eG95N3B1UnFpQ3puQmR3ZDVJYl83WkgwWERKSGtFQkJ1RHE0NW1oTzY4UW02cjk0Z3huU05qQXY1ZA==",
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0GAAAAAAAAAJ4AAAAAAAAAMCwtdnY4X2Y0QUFRSURCQVVHQndnSkNnc01EUTRQRUJFU3h0Z2kydWFXZ2xNeGFLV2JTLTUxNkZyNDNobE5kX2NPVjIzdTczaklYT2I1RldOTUFrOFluTDBqY043UEg1YWtUUXBwYnF3a2ltM2s2N29mZjNSbU1nQWd3b1J0VzZLdGk0eklfZTNjdWF3TzN1eWNKWDNfaE4tNWlWOEmeAAAAAAAAADEsRXhRVkZoY1lHUm9iSEIwZUh5QWhJaU1rSlNZbktDa3F3Rkhkc29QZjNQWU50UzRkM0xnejZQaDl6Y3NrZ1pnbG9fOWhTZXFzb3RMWjNYRGlXbzhucmtEblZKTmRjVy1pR1k3aEFudE1BN0NBcnZNeEV2U2RwYXZLanJlV3l6Wktrc0ZralBBMEU0dUdTdjdrbEVjUW9LbHRFQjJKngAAAAAAAAAyLEt5d3RMaTh3TVRJek5EVTJOemc1T2pzOFBUNF9RRUZDQzA3dzBTTUo0eTVzQTJkVERCYXEwdV9nc3pMcXJUX0lKNHVDUjZkV2Vhb0tnU2hZTUZ2SHM0ZE5QLW1TS3h2WE1PZmJPZHNFaUQxMEltOWhSeDFvRnJLMUFpNVljS18xWnJkYkw0ZC1TUWpXVmNkcE5PZWpQZHJjRlVoN54AAAAAAAAAMyxRMFJGUmtkSVNVcExURTFPVDFCUlVsTlVWVlpYV0ZsYWpUZU1BMl9KZUxObGpZRFl3MExYeU9idlNmOHR0VXBlSm8tTzNPeUtMR1lhZ1FRd1ZjUF9RR1RoSEN1NGdlSWtBbUxCMkZhSU9GTDR0cTV4TzcweTc2ZDdXOUJFWG1wcjlLZTBSNExNaVRucVhhQWpNaFJTaDJ0TEVPUFqeAAAAAAAAADQsVzF4ZFhsOWdZV0pqWkdWbVoyaHBhbXRzYlc1dmNIRnlmTTdWTGNyYVUtbjNTMk9LX0hXZmtFTVp0WEZ5UkdNRmJKQ3ZGaFJfSlFJSFNwTFF4UUw3SHNMNklCTTRmVHVBcTY3OG9wbUlPRDB2d0UwRndvTTlKcjhLenhnSUlIcDBuV1ExM1BwTmlJMU8wblNCSG5Fd01aLVhZWThYngAAAAAAAAA1LGMzUjFkbmQ0ZVhwN2ZIMS1mNENCZ29PRWhZYUhpSW1LX1g4S3ZhUWtFZnhHR01wSTZLcVM1dDA4SERtTTdzQlllT3NEdDRaODd4Y0x1dkY1MWRtY3ktbVBEdUZvMnJBeUhiU0N0RE5uMWRjZk1QNkJhbG1hUjF3blRqdjU1WW1iSWhDTzlqcHp5bFVKR09rYTNRUE9rdlZvX3BwUA==",
		"MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0GAAAAAAAAAJ4AAAAAAAAAMCwtdnY4X2Y0QUFRSURCQVVHQndnSkNnc01EUTRQRUJFU3h0Z2kydWFXZ2xNeGFLV2JTLTUxNkZyNDNobE5kX2NPVjIzdTczaklYT2I1RldOTUFrOFluTDBqY05EUEQ0cjZIUmRwRk84ZW5tU3p0THBIQ2xGS1BpUUJ0Yk5UZW9uZDU1T0lSY0NTQmR0Q2wxRngzNVNLWlVoUHF3PT2eAAAAAAAAADEsRXhRVkZoY1lHUm9iSEIwZUh5QWhJaU1rSlNZbktDa3F3Rkhkc29QZjNQWU50UzRkM0xnejZQaDl6Y3NrZ1pnbG9fOWhTZXFzb3RMWjNYRGlXbzhucmtEblZKMWRZWFBfU1pEaGVEaDJGN25YOGZOcFo5R3hxWV9yLVlDbzZpczZ5TjRrdklMMkMxVlVhSVBBemZ5VXVhMWJnZz09ngAAAAAAAAAyLEt5d3RMaTh3TVRJek5EVTJOemc1T2pzOFBUNF9RRUZDQzA3dzBTTUo0eTVzQTJkVERCYXEwdV9nc3pMcXJUX0lKNHVDUjZkV2Vhb0tnU2hZTUZ2SHM0ZE5QLWVTT3dlTFlQamJRNWctbkRRamZXODVNamhFR3BhVWRSbG1VYUtGTEtnYmVUSjFsNE1GdTE3TVpqMy1ramdCbUE9PZ4AAAAAAAAAMyxRMFJGUmtkSVNVcExURTFPVDFCUlVsTlVWVlpYV0ZsYWpUZU1BMl9KZUxObGpZRFl3MExYeU9idlNmOHR0VXBlSm8tTzNPeUtMR1lhZ1FRd1ZjUF9RR1RoSENXNGtmNV9VbnJCb2hXeUxGdXY2YTRwVHBnZTQ0TmFMT2Q2ZkZFWWlMajAzRHVtNC1ZSUM5VGRnN2JzME13N2h3PT2eAAAAAAAAADQsVzF4ZFhsOWdZV0pqWkdWbVoyaHBhbXRzYlc1dmNIRnlmTTdWTGNyYVUtbjNTMk9LX0hXZmtFTVp0WEZ5UkdNRmJKQ3ZGaFJfSlFJSFNwTFF4UUw3SHNMNklCMDRiU2ZhLTdmODJOcXlMRFI0bjAxZHQ2WVJLcHNydUM4MkFsRUg4WHQxUGhQb1ZmX2FJT2V4NmNOSFhUQ0hIQT09ngAAAAAAAAA1LGMzUjFkbmQ0ZVhwN2ZIMS1mNENCZ29PRWhZYUhpSW1LX1g4S3ZhUWtFZnhHR01wSTZLcVM1dDA4SERtTTdzQlllT3NEdDRaODd4Y0x1dkY1MWRtY3ktbVBEdTlveXF4dFRhaUN6bkJkd2Q1SWJfN1pIM3kyUzNnR09Rekh4TExyWGdfT3JGSWFtMlBmdWJqanlWRjMyWDVMdmc9PQ==",
	}

	syncPath := baseDir + "collector/"
	txLog := makeTransactionLog(syncPath, password, t)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	remoteKv, err := NewOrLoadRemoteKv(txLog, kv, nil, nil, nil)
	require.NoError(t, err)

	myId := "testingMyId"

	workingDir := baseDir + "remoteFsSmoke/"
	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	fsRemote := NewFileSystemRemoteStorage(workingDir)

	devices := make([]string, 0)
	for i, remoteTxLogEnc := range remoteTxLogsEnc {
		mockDeviceId := strconv.Itoa(i)
		mockTxLog, err := base64.StdEncoding.DecodeString(remoteTxLogEnc)
		require.NoError(t, err)
		require.NoError(t, fsRemote.Write(mockDeviceId, mockTxLog))
		devices = append(devices, mockDeviceId)
	}

	collector := NewCollector(syncPath, myId, txLog, fsRemote, remoteKv)

	txChanges, newUpdates, err := collector.collectChanges(devices)
	require.NoError(t, err)

	// Ensure all new updates are today
	for _, newUpdate := range newUpdates {
		require.Equal(t, newUpdate.Day(), time.Now().Day())
	}

	// Ensure all txChanges have expected values
	for _, txChange := range txChanges {
		require.Equal(t, txChange[0].Timestamp.Year(), 2053)
	}

}

// Unit test for ReadTransactionAfter.
func TestReadTransactionAfter(t *testing.T) {

	// Note: these are pre-canned serialized transaction logs w/ transactions
	// with timestamp values in the year 2053
	const txSerialEnd = "MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0GAAAAAAAAAJIAAAAAAAAAMCwtdnY4X2Y0QUFRSURCQVVHQndnSkNnc01EUTRQRUJFU3h0Z2kydWFXZ2xNeGFLV2JTLTUxNkZyNDNobE5kX2NPVjIzdTczaklYT2I1RldOTUFrOFluTDBqY05fRkZjLTNRd1FkV2FFOW1pcjg4X3dRYmtGQU9XNWYwSmhhZkR5YzZWQmdXUldQYUx5N2Y1YUiSAAAAAAAAADEsRXhRVkZoY1lHUm9iSEIwZUh5QWhJaU1rSlNZbktDa3F3Rkhkc29QZjNQWU50UzRkM0xnejZQaDl6Y3NrZ1pnbG9fOWhTZXFzb3RMWjNYRGlXbzhucmtEblZKSlhleld5RjRDVk5YWlZFX2VZdHJVLUE4RzdtTVcxbkt0d2JST2JCN1dfODhRaVFxX3JTU1o2kgAAAAAAAAAyLEt5d3RMaTh3TVRJek5EVTJOemc1T2pzOFBUNF9RRUZDQzA3dzBTTUo0eTVzQTJkVERCYXEwdV9nc3pMcXJUX0lKNHVDUjZkV2Vhb0tnU2hZTUZ2SHM0ZE5QLWlZSVVER1B1bXZEdFlkbUhwc09pbHVWaWhPTzl6S0VES1FiNVk4WWk5a1hmWlhwdFUyYVgtZ5IAAAAAAAAAMyxRMFJGUmtkSVNVcExURTFPVDFCUlVsTlVWVlpYV0ZsYWpUZU1BMl9KZUxObGpZRFl3MExYeU9idlNmOHR0VXBlSm8tTzNPeUtMR1lhZ1FRd1ZjUF9RR1RoSENxeWk3NHlER3kxNzF1UktCWGdydWgtS29nWDlNa0VTY3lBdWNKRXJhVDEwMUxlQWFDRW1rTHqSAAAAAAAAADQsVzF4ZFhsOWdZV0pqWkdWbVoyaHBhbXRzYlc1dmNIRnlmTTdWTGNyYVUtbjNTMk9LX0hXZmtFTVp0WEZ5UkdNRmJKQ3ZGaFJfSlFJSFNwTFF4UUw3SHNMNklCSXlkMmFYcGFDSWxaU1JLSG8zMkFzSzA3WVlMZEYxM1FTbVpaTWNjUFgtSERkd1dDVm13VmFpkgAAAAAAAAA1LGMzUjFkbmQ0ZVhwN2ZIMS1mNENCZ29PRWhZYUhpSW1LX1g4S3ZhUWtFZnhHR01wSTZLcVM1dDA4SERtTTdzQlllT3NEdDRaODd4Y0x1dkY1MWRtY3ktbVBEdUJpME9nZ0U3cjJnejUteFpBSEtMaU9lMnk4WERKWVhDZm5uNUZwb1Q5WTB3Nk1LLWVsTjQzWA=="

	// Decode serialized value
	txSerial, err := base64.StdEncoding.DecodeString(txSerialEnd)
	require.NoError(t, err)

	txLog := TransactionLog{
		deviceSecret: []byte("deviceSecret"),
	}
	require.NoError(t, txLog.deserialize(txSerial))

	// Create timestamp after txLog's timestamps (in 2053)
	lateTimestamp, err := time.Parse(time.RFC3339,
		"3000-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Read transaction after late timestamp
	received, err := readTransactionsAfter(
		txSerial, lateTimestamp, txLog.deviceSecret)
	require.NoError(t, err)

	// Ensure no transactions exist
	require.Len(t, received, 0)

	// Create timestamp before transactions in txLog
	earlyTimestamp, err := time.Parse(time.RFC3339,
		"2000-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Read transaction after early timestamp
	received, err = readTransactionsAfter(
		txSerial, earlyTimestamp, txLog.deviceSecret)
	require.NoError(t, err)

	// Ensure it has all the transactions
	require.Len(t, received, len(txLog.txs))

}

func makeCollector(t *testing.T) *Collector {
	syncPath := baseDir + "collector/"
	txLog := makeTransactionLog(syncPath, password, t)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	remoteKv, err := NewOrLoadRemoteKv(txLog, kv, nil, nil, nil)
	require.NoError(t, err)

	myId := "testingMyId"

	workingDir := baseDir + "remoteFsSmoke/"
	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	fsRemote := NewFileSystemRemoteStorage(workingDir)

	return NewCollector(syncPath, myId, txLog, fsRemote, remoteKv)
}
