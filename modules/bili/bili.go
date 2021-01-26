package bili

import (
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
)

const ModuleName = "bili"

var instance *bili
var logger = utils.GetModuleLogger(ModuleName)

func init() {
	instance = NewBili()
	bot.RegisterModule(instance)
}
