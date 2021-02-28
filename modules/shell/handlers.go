package shell

import (
	"fmt"
	"github.com/zhouziqunzzq/MiraiGo-DD/modules/bili"
	suki "github.com/zhouziqunzzq/MiraiGo-DD/modules/daredemo_suki"
	"github.com/zhouziqunzzq/MiraiGo-DD/modules/naive_chatbot"
	"strconv"
	"strings"
)
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

func handleDd(ctx *CmdContext) {
	if originMsg, ok := ctx.OriginMsg.(*message.GroupMessage); ok {
		suki.SendDdPic(ctx.Client, originMsg.GroupCode)
	}
}

func handleLs(ctx *CmdContext) {
	if len(ctx.ParsedCmd.Args) == 0 {
		sendTextRsp("参数错误", ctx)
	} else {
		switch ctx.ParsedCmd.Args[0] {
		case "bili":
			if gMsg, ok := ctx.OriginMsg.(*message.GroupMessage); ok {
				userInfoList := bili.GetSubscriptionByGroupId(gMsg.GroupCode)
				if userInfoList == nil || len(userInfoList) == 0 {
					sendTextRsp("暂无订阅的主播", ctx)
				} else {
					sb := strings.Builder{}
					sb.WriteString("当前订阅的主播信息如下：\n")
					for _, userInfo := range userInfoList {
						if len(userInfo.Name) == 0 {
							sb.WriteString(fmt.Sprintf("UID: %d", userInfo.Mid))
						} else {
							sb.WriteString(fmt.Sprintf("UID: %d - %s - ", userInfo.Mid, userInfo.Name))
							if userInfo.LiveRoom.LiveStatus == bili.Streaming {
								sb.WriteString("已开播")
							} else {
								sb.WriteString("未开播")
							}
						}
						sb.WriteRune('\n')
					}
					sb.WriteString("（注：信息拉取存在延时，未显示主播昵称表明尚未拉取，请稍后重试）")
					sendTextRsp(sb.String(), ctx)
				}
			} else {
				sendTextRsp("暂时仅支持群内查询订阅主播信息", ctx)
			}
		case "chatbot":
			if _, ok := ctx.OriginMsg.(*message.GroupMessage); ok {
				triggerProb, err := naive_chatbot.GetTriggerProb()
				if err != nil {
					sendTextRsp("聊天机器人未启用", ctx)
				} else {
					rsp := "聊天机器人已启用\n"
					rsp += fmt.Sprintf("触发概率：%f", triggerProb)
					sendTextRsp(rsp, ctx)
				}
			}
		default:
			sendTextRsp(fmt.Sprintf("对象%s暂不支持 ls 命令", ctx.ParsedCmd.Args[0]), ctx)
		}
	}
}

func handleSet(ctx *CmdContext) {
	if len(ctx.ParsedCmd.Args) == 0 {
		sendTextRsp("参数错误", ctx)
	} else {
		switch ctx.ParsedCmd.Args[0] {
		case "chatbot":
			// set chatbot <key> <value>
			if len(ctx.ParsedCmd.Args) < 3 {
				sendTextRsp("参数错误！命令格式：set chatbot <key> <value>", ctx)
				return
			}
			switch ctx.ParsedCmd.Args[1] {
			case "trigger_prob":
				newProb, err := strconv.ParseFloat(ctx.ParsedCmd.Args[2], 32)
				if err != nil {
					sendTextRsp(fmt.Sprintf("无效的 value (%s): %v", ctx.ParsedCmd.Args[2], err), ctx)
					return
				}
				err = naive_chatbot.SetTriggerProb(float32(newProb))
				if err != nil {
					sendTextRsp(fmt.Sprintf("无效的 value (%s): %v", ctx.ParsedCmd.Args[2], err), ctx)
					return
				}
				sendTextRsp("参数更新成功", ctx)
			default:
				sendTextRsp(fmt.Sprintf("无效的 key (%s)", ctx.ParsedCmd.Args[1]), ctx)
			}
		default:
			sendTextRsp(fmt.Sprintf("对象%s暂不支持 set 命令", ctx.ParsedCmd.Args[0]), ctx)
		}
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
