package naive_chatbot

import (
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
)

const ModuleName = "naive_chatbot"

var instance *chatbot
var logger = utils.GetModuleLogger(ModuleName)

func init() {
	instance = NewChatbot()
	bot.RegisterModule(instance)
}
