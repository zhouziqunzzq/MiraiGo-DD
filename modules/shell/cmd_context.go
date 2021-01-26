package shell

import "github.com/Mrs4s/MiraiGo/client"

type CmdContext struct {
	ParsedCmd *ParsedCmd
	Client    *client.QQClient
	OriginMsg interface{}
}

func NewCmdContext(parsedCmd *ParsedCmd, client *client.QQClient, originMsg interface{}) *CmdContext {
	return &CmdContext{
		ParsedCmd: parsedCmd,
		Client:    client,
		OriginMsg: originMsg,
	}
}
