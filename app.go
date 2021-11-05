package main

import (
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/config"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
	"os"
	"os/signal"

	// register modules
	// not sorted alphabetically as intended
	_ "github.com/zhouziqunzzq/MiraiGo-DD/modules/auto_reconnect"
	_ "github.com/zhouziqunzzq/MiraiGo-DD/modules/logging"
	_ "github.com/zhouziqunzzq/MiraiGo-DD/modules/shell"
	_ "github.com/zhouziqunzzq/MiraiGo-DD/modules/bili"
	_ "github.com/zhouziqunzzq/MiraiGo-DD/modules/daredemo_suki"
	_ "github.com/zhouziqunzzq/MiraiGo-DD/modules/naive_chatbot"
	_ "github.com/zhouziqunzzq/MiraiGo-DD/modules/diary"
)

func init() {
	utils.WriteLogToFS()
	config.Init()
}

func main() {
	// Generate random device.json if necessary
	bot.GenRandomDevice()

	// 快速初始化
	bot.Init()

	// 初始化 Modules
	bot.StartService()

	// 使用协议
	// 不同协议可能会有部分功能无法使用
	// 在登陆前切换协议
	bot.UseProtocol(bot.AndroidPhone)

	// 登录
	bot.Login()

	// 刷新好友列表，群列表
	bot.RefreshList()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch
	bot.Stop()
}
