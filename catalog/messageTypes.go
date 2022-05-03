package catalog

type MessageType uint32

const MessageTypeLen = 32 / 8

const (
	/*general message types*/

	// NoType - Used as a wildcard for listeners to listen to all existing types.
	// Think of it as "No type in particular"
	NoType MessageType = 0

	// XxMessage - Type of message sent by the xx messenger.
	XxMessage = 2

	/*End to End Rekey message types*/

	// KeyExchangeTrigger - Trigger a rekey, this message is used locally in
	// client only
	KeyExchangeTrigger = 30
	// KeyExchangeConfirm - Rekey confirmation message. Sent by partner to
	//confirm completion of a rekey
	KeyExchangeConfirm = 31

	// KeyExchangeTriggerEphemeral - Trigger a rekey, this message is used
	//locally in client only. For ephemeral only e2e instances.
	KeyExchangeTriggerEphemeral = 32
	// KeyExchangeConfirmEphemeral - Rekey confirmation message. Sent by partner
	// to confirm completion of a rekey. For ephemeral only e2e instances.
	KeyExchangeConfirmEphemeral = 33

	/* Group chat message types */

	// GroupCreationRequest - A group chat request message sent to all members in a group.
	GroupCreationRequest = 40

	// NewFileTransfer is transmitted first on the initialization of a file
	// transfer to inform the receiver about the incoming file.
	NewFileTransfer = 50

	// EndFileTransfer is sent once all file parts have been transmitted to
	// inform the receiver that the file transfer has ended.
	EndFileTransfer = 51

	// SimpleStream - e2e streaming message
	SimpleStream = 60
)
