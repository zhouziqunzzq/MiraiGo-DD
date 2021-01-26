package shell

import "github.com/Mrs4s/MiraiGo/message"

func handlePing(ctx *CmdContext) {
	sendTextRsp("pong", ctx)
}

func handlePersecute(ctx *CmdContext) {
	if len(ctx.ParsedCmd.Args) != 1 {
		sendTextRsp("请提供唯一的参数", ctx)
	} else {
		sendTextRsp("今天也在迫害"+ctx.ParsedCmd.Args[0]+"嘛", ctx)
	}
}

func sendTextRsp(rsp string, ctx *CmdContext) {
	rspMsg := message.NewSendingMessage()
	rspMsg.Append(message.NewText(rsp))

	qqClient := ctx.Client
	switch originMsg := ctx.OriginMsg.(type) {
	case *message.PrivateMessage:
		qqClient.SendPrivateMessage(originMsg.Sender.Uin, rspMsg)
	case *message.GroupMessage:
		qqClient.SendGroupMessage(originMsg.GroupCode, rspMsg)
	default:
		logger.Warnf("unhandled origin msg type for outgoing msg: %s", rsp)
	}
}
