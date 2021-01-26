package shell

import (
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/config"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
	"gopkg.in/yaml.v2"
	"sync"
)

type shell struct {
	isEnabled        bool
	config           Config
	adminIdMap       map[int64]bool
	cmdHandlerMap    map[string]func(*CmdContext)
	cmdCheckAdminMap map[string]bool
}

func NewShell() *shell {
	return &shell{
		isEnabled:        false,
		config:           Config{},
		adminIdMap:       make(map[int64]bool),
		cmdHandlerMap:    make(map[string]func(ctx *CmdContext)),
		cmdCheckAdminMap: make(map[string]bool),
	}
}

func (m *shell) MiraiGoModule() bot.ModuleInfo {
	return bot.ModuleInfo{
		ID:       ModuleName,
		Instance: instance,
	}
}

func (m *shell) Init() {
	// check is_enabled
	m.isEnabled = config.GlobalConfig.GetBool("modules." + ModuleName + ".is_enabled")
	if !m.isEnabled {
		logger.Info("this module is disabled by global config")
		return
	}

	// load module config
	configPath := config.GlobalConfig.GetString("modules." + ModuleName + ".config_path")
	if configPath == "" {
		configPath = "./shell.yaml"
	}
	logger.Debugf("reading config from %s", configPath)
	cb := utils.ReadFile(configPath)
	err := yaml.Unmarshal(cb, &m.config)
	if err != nil {
		logger.WithError(err).Errorf("unable to read config file in %s", configPath)
		m.isEnabled = false
		return
	}

	// load admin id list
	// TODO: grant admin role as per Group, not just to User ID
	for _, groupCode := range m.config.AdminIdList {
		m.adminIdMap[groupCode] = true
		logger.Infof("admin enabled for ID %d", groupCode)
	}

	// register cmd handlers
	m.registerCmd("ping", handlePing, false)
	m.registerCmd("persecute", handlePersecute, true)
	m.registerCmd("dd", handleDd, false)
}

func (m *shell) PostInit() {
	// 第二次初始化
	// 再次过程中可以进行跨Module的动作
	// 如通用数据库等等
}

func (m *shell) Serve(b *bot.Bot) {
	if m.isEnabled {
		m.registerCallbacks(b)
	}
}

func (m *shell) Start(b *bot.Bot) {
	// 此函数会新开携程进行调用
	// ```go
	// 		go exampleModule.Start()
	// ```

	// 可以利用此部分进行后台操作
	// 如http服务器等等
}

func (m *shell) Stop(b *bot.Bot, wg *sync.WaitGroup) {
	// 别忘了解锁
	defer wg.Done()
	// 结束部分
	// 一般调用此函数时，程序接收到 os.Interrupt 信号
	// 即将退出
	// 在此处应该释放相应的资源或者对状态进行保存
}

func (m *shell) checkAdminPermission(cmdName string, userId int64) bool {
	if v, ok := m.cmdCheckAdminMap[cmdName]; !ok || !v {
		return true
	}
	if _, ok := m.adminIdMap[userId]; ok {
		return true
	}
	return false
}

func (m *shell) registerCmd(name string, handler func(*CmdContext), needAdminCheck bool) {
	if _, ok := m.cmdHandlerMap[name]; ok {
		logger.Warnf("command %s already exist", name)
		return
	}

	m.cmdHandlerMap[name] = handler
	if needAdminCheck {
		m.cmdCheckAdminMap[name] = true
	}
}

func (m *shell) getParseErrResp() *message.SendingMessage {
	msg := message.NewSendingMessage()
	msg.Append(message.NewText("命令格式错误，请检查后重试～"))
	return msg
}

func (m *shell) getCmdNotFoundErrResp() *message.SendingMessage {
	msg := message.NewSendingMessage()
	msg.Append(message.NewText("不支持的命令，请检查后重试～"))
	return msg
}

func (m *shell) getUnauthorizedErrResp() *message.SendingMessage {
	msg := message.NewSendingMessage()
	msg.Append(message.NewText("您的权限不足 QAQ"))
	return msg
}

func (m *shell) handleGroupMessage(qqClient *client.QQClient, groupMessage *message.GroupMessage) {
	rawStr := groupMessage.ToString()
	// parse cmd
	parsedCmd, err := parseCmd(rawStr)
	if parsedCmd == nil {
		logger.Debugf("not a cmd, skipping group message %s", rawStr)
		return
	}
	if err != nil {
		logger.WithError(err).Errorf("failed to parse cmd %s", rawStr)
		qqClient.SendGroupMessage(groupMessage.GroupCode, m.getParseErrResp())
		return
	}

	// try to handle cmd
	if handler, ok := m.cmdHandlerMap[parsedCmd.Name]; ok {
		// check admin
		if m.checkAdminPermission(parsedCmd.Name, groupMessage.Sender.Uin) {
			// build context
			ctx := NewCmdContext(parsedCmd, qqClient, groupMessage)
			// handle cmd async
			go handler(ctx)
		} else {
			logger.WithError(err).Debugf(
				"unauthorized user %d try to access cmd %s",
				groupMessage.Sender.Uin,
				rawStr,
			)
			qqClient.SendGroupMessage(groupMessage.GroupCode, m.getUnauthorizedErrResp())
		}
	} else {
		logger.WithError(err).Debugf("cmd %s not found", rawStr)
		qqClient.SendGroupMessage(groupMessage.GroupCode, m.getCmdNotFoundErrResp())
	}
}

func (m *shell) registerCallbacks(b *bot.Bot) {
	b.OnGroupMessage(m.handleGroupMessage)
}
