package shell

import (
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
)

const ModuleName = "shell"

var instance *shell
var logger = utils.GetModuleLogger(ModuleName)

func init() {
	instance = NewShell()
	bot.RegisterModule(instance)
}
