import sys

whitelist = {
    "field": ["AdminKeysUpdateJSON.ChannelId", 
              "AuthenticatedConnection.Connection", 
              "BackupReport.RestoredContacts", "ChannelSendReport.RoundsList",
              "ChannelUpdateJSON.ChannelID", "DmBlockedUserJSON.User",
              "DmMessageDeletedJSON.MessageID", "DmMessageReceivedJSON.UUID",
              "DmMessageReceivedJSON.PubKey",
              "DmNotificationUpdateJSON.Changed",
              "DmNotificationUpdateJSON.Deleted",
              "DmNotificationUpdateJSON.NotificationFilter",
              "DmTokenUpdateJSON.ChannelId",
              "NotificationUpdateJSON.ChangedNotificationStates",
              "NotificationUpdateJSON.DeletedNotificationStates",
              "NotificationUpdateJSON.NotificationFilters",
              "NotificationUpdateJSON.MaxState", "Progress.TransferID",
              "ReceivedChannelMessageReport.RoundsList", 
              "ReceivedFile.TransferID", "ReceivedFile.SenderID",
              "RestlikeMessage.Version", "RoundsList.Rounds",
              "SingleUseCallbackReport.RoundsList",
              "SingleUseCallbackReport.Partner",
              "SingleUseCallbackReport.ReceptionID",
              "SingleUseResponseReport.RoundsList",
              "SingleUseResponseReport.ReceptionID",
              "SingleUseSendReport.RoundsList", 
              "SingleUseSendReport.ReceptionID",
              "UserMutedJSON.ChannelID", "UserMutedJSON.PubKey",
              "E2ESendReport.RoundsList", "FtReceivedProgress.ID",
              "GroupReport.RoundsList", "GroupSendReport.RoundsList",
              "FtSentProgress.ID", "MessageDeletedJSON.MessageID",
              "MessageReceivedJSON.ChannelID", "NickNameUpdateJSON.ChannelId"],
    "constructor": ["ChannelsManager.NewChannelsManagerGoEventModel",
                    "DMClient.NewDMClientWithGoEventModel"],
    "method": [],
    "function": ["GetCMixInstance", "LoadChannelsManagerGoEventModel",
                 "NewChannelsManagerGoEventModel", 
                 "NewDMClientWithGoEventModel"]
}

failure = False

with open(sys.argv[1], "r", encoding="UTF-8") as f:
    for line in f.readlines():
        if line.startswith("// skipped "):
            parts = line.split(" ", 4)
            
            type = parts[2]
            object = parts[3]
            reason = parts[4]

            if object not in whitelist[type]:
                print(line.strip())
                failure = True

if failure:
    sys.exit(-1)