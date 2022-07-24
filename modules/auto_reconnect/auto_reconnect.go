package auto_reconnect

import (
	"sync"
	"syscall"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
)

func init() {
	instance = &autoReconnect{}
	bot.RegisterModule(instance)
}

type autoReconnect struct{}

var instance *autoReconnect
var logger = utils.GetModuleLogger("internal.autoReconnect")

func (m *autoReconnect) MiraiGoModule() bot.ModuleInfo {
	return bot.ModuleInfo{
		ID:       "internal.auto_reconnect",
		Instance: instance,
	}
}

func (m *autoReconnect) Init() {}

func (m *autoReconnect) PostInit() {}

func (m *autoReconnect) Serve(b *bot.Bot) {
	registerAutoReconnect(b)
}

func (m *autoReconnect) Start(b *bot.Bot) {}

func (m *autoReconnect) Stop(b *bot.Bot, wg *sync.WaitGroup) {
	defer wg.Done()
}

func registerAutoReconnect(b *bot.Bot) {
	b.DisconnectedEvent.Subscribe(func(qqClient *client.QQClient, event *client.ClientDisconnectedEvent) {
		// try to reconnect
		cnt := 0
		for cnt < 10 {
			cnt++
			if rsp, err := b.Login(); err != nil || !rsp.Success {
				logger.Warnf("reconnect failed, retrying (%d/%d)", cnt, 10)
				continue
			} else {
				// reconnect success, refresh info
				logger.Info("reconnect success")
				bot.RefreshList()
				break
			}
		}

		// terminate if reconnection failed
		if !b.Online.Load() {
			logger.Error("failed to restore from disconnection, exiting")
			_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		}
	})
}
