Channels provides a channels implementation on top of broadcast which is capable of handing the user facing features of
channels, including replies, reactions, and eventually admin commands.

on sending, data propagates as follows:
Send function (Example: SendMessage) - > SendGeneric ->
Broadcast.BroadcastWithAssembler -> cmix.SendWithAssembler

on receiving messages propagate as follows:
cmix message pickup (by service)- > broadcast.Processor ->
userListener ->  events.triggerEvent ->
messageTypeHandler (example: Text) ->
eventModel (example: ReceiveMessage)

on sendingAdmin, data propagates as follows:
Send function - > SendAdminGeneric ->
Broadcast.BroadcastAsymmetricWithAssembler -> cmix.SendWithAssembler

on receiving admin messages propagate as follows:
cmix message pickup (by service)- > broadcast.Processor -> adminListener ->
events.triggerAdminEvent -> messageTypeHandler (example: Text) ->
eventModel (example: ReceiveMessage)