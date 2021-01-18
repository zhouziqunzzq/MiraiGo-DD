package daredemo_suki

import (
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
)

const ModuleName = "daredemo_suki"

var instance *suki
var logger = utils.GetModuleLogger(ModuleName)

func init() {
	instance = NewSuki()
	bot.RegisterModule(instance)
}
