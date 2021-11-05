package diary

import (
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
)

const ModuleName = "diary"

var instance *diary
var logger = utils.GetModuleLogger(ModuleName)

func init() {
	instance = NewDiary()
	bot.RegisterModule(instance)
}
