package daredemo_suki

import "github.com/Mrs4s/MiraiGo/client"

// These are APIs exposed to other modules.
// They should only be called after initialization of all modules.

func SendDdPic(qqClient *client.QQClient, groupId int64) {
	if instance != nil {
		instance.SendDdPic(qqClient, groupId)
	}
}
