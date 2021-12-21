///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

const TypeLen = 4

type Type uint32

const (
	/*general message types*/
	// Used as a wildcard for listeners to listen to all existing types.
	// Think of it as "No type in particular"
	NoType Type = 0

	// A message with no message structure
	// this is a reserved type, messages sent via SendCmix automatically gain
	// this type. Sent messages with this type will be rejected and received
	// non Cmix messages will be ignored
	Raw Type = 1

	//General text message, contains human readable text
	Text Type = 2

	/*User Discovery message types*/
	//Message structures defined in the UD package

	// A search for users based on facts.  A series of hashed facts are passed
	// to UDB
	UdSearch = 10

	// The response to the UD search. It contains a list of contact objects
	// matching the sent facts
	UdSearchResponse = 11

	// Searched for the DH public key associated with the passed User ID
	UdLookup = 12

	// Response to UdLookup, it contains the associated public key if one is
	// available
	UdLookupResponse = 13

	/*End to End Rekey message types*/
	// Trigger a rekey, this message is used locally in client only
	KeyExchangeTrigger = 30
	// Rekey confirmation message. Sent by partner to confirm completion of a rekey
	KeyExchangeConfirm = 31

	/* Group chat message types */
	// A group chat request message sent to all members in a group.
	GroupCreationRequest = 40

	// NewFileTransfer is transmitted first on the initialization of a file
	// transfer to inform the receiver about the incoming file.
	NewFileTransfer = 50

	// EndFileTransfer is sent once all file parts have been transmitted to
	// inform the receiver that the file transfer has ended.
	EndFileTransfer = 51
)
