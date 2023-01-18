////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"time"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

type mockClient struct{}

func (mc *mockClient) GetMaxMessageLength() int {
	return 2048
}
func (mc *mockClient) SendManyWithAssembler(recipients []*id.ID,
	assembler cmix.ManyMessageAssembler,
	params cmix.CMIXParams) (rounds.Round, []ephemeral.Id, error) {
	return rounds.Round{}, nil, nil
}
func (mc *mockClient) AddIdentity(*id.ID, time.Time, bool,
	message.Processor) {
}
func (mc *mockClient) AddIdentityWithHistory(*id.ID, time.Time, time.Time,
	bool, message.Processor) {
}
func (mc *mockClient) AddService(*id.ID, message.Service,
	message.Processor) {
}
func (mc *mockClient) DeleteClientService(*id.ID) {}
func (mc *mockClient) RemoveIdentity(*id.ID)      {}
func (mc *mockClient) GetRoundResults(time.Duration, cmix.RoundEventCallback,
	...id.Round) {
}
func (mc *mockClient) AddHealthCallback(func(bool)) uint64 { return 0 }
func (mc *mockClient) RemoveHealthCallback(uint64)         {}
