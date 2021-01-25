package daredemo_suki

import (
	"bytes"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/config"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math/rand"
	"path"
	"strings"
	"sync"
)

type suki struct {
	isEnabled        bool
	config           SukiConfig
	enabledGroupsMap map[int64]bool
	ddImgPool        [][]byte
}

func NewSuki() *suki {
	return &suki{
		isEnabled:        false,
		config:           SukiConfig{},
		enabledGroupsMap: make(map[int64]bool),
		ddImgPool:        make([][]byte, 0),
	}
}

func (m *suki) MiraiGoModule() bot.ModuleInfo {
	return bot.ModuleInfo{
		ID:       ModuleName,
		Instance: instance,
	}
}

func (m *suki) Init() {
	// check is_enabled
	m.isEnabled = config.GlobalConfig.GetBool("modules." + ModuleName + ".is_enabled")
	if !m.isEnabled {
		logger.Info("this module is disabled by global config")
		return
	}

	// load module config
	configPath := config.GlobalConfig.GetString("modules." + ModuleName + ".config_path")
	if configPath == "" {
		configPath = "./dd.yaml"
	}
	logger.Debugf("reading config from %s", configPath)
	bytes := utils.ReadFile(configPath)
	err := yaml.Unmarshal(bytes, &m.config)
	if err != nil {
		logger.WithError(err).Errorf("unable to read config file in %s", configPath)
		m.isEnabled = false
		return
	}

	// load enabled groups
	for _, groupCode := range m.config.EnabledGroups {
		m.enabledGroupsMap[groupCode] = true
		logger.Infof("DD enabled for group %d", groupCode)
	}

	// load dd img
	files, err := ioutil.ReadDir(m.config.ImgPath)
	if err != nil {
		logger.WithError(err).Errorf("unable to load img from %s", m.config.ImgPath)
		m.isEnabled = false
		return
	}
	if len(files) == 0 {
		logger.Errorf("no file found in %s", m.config.ImgPath)
		m.isEnabled = false
		return
	}
	for _, imgFile := range files {
		m.ddImgPool = append(m.ddImgPool, utils.ReadFile(
			path.Join(m.config.ImgPath, imgFile.Name()),
		))
	}
	logger.Debugf("%d img files loaded in %s", len(files), m.config.ImgPath)
}

func (m *suki) PostInit() {
	// 第二次初始化
	// 再次过程中可以进行跨Module的动作
	// 如通用数据库等等
}

func (m *suki) Serve(b *bot.Bot) {
	if m.isEnabled {
		m.registerCallbacks(b)
	}
}

func (m *suki) Start(b *bot.Bot) {
	// 此函数会新开携程进行调用
	// ```go
	// 		go exampleModule.Start()
	// ```

	// 可以利用此部分进行后台操作
	// 如http服务器等等
}

func (m *suki) Stop(b *bot.Bot, wg *sync.WaitGroup) {
	// 别忘了解锁
	defer wg.Done()
	// 结束部分
	// 一般调用此函数时，程序接收到 os.Interrupt 信号
	// 即将退出
	// 在此处应该释放相应的资源或者对状态进行保存
}

func (m *suki) checkKeywords(s string) bool {
	for _, w := range m.config.Keywords {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func (m *suki) handleGroupMessage(qqClient *client.QQClient, groupMessage *message.GroupMessage) {
	// filter enabled groups
	if _, ok := m.enabledGroupsMap[groupMessage.GroupCode]; !ok {
		logger.Debugf("ignoring group message from group chat %s(%d)",
			groupMessage.GroupName,
			groupMessage.GroupCode,
		)
		return
	}

	// check keywords
	performDD := false
	for _, elem := range groupMessage.Elements {
		if elem.Type() == message.Text {
			msg := elem.(*message.TextElement)
			if m.checkKeywords(msg.Content) {
				performDD = true
				break
			}
		}
	}
	if !performDD {
		return
	}

	// send random DD meme img
	logger.Infof("DD triggered by message: %s", groupMessage.ToString())
	msg := message.NewSendingMessage()
	selectedImg := m.ddImgPool[rand.Intn(len(m.ddImgPool))]
	r := bytes.NewReader(selectedImg)
	upImg, err := qqClient.UploadGroupImage(groupMessage.GroupCode, r)
	if err != nil {
		logger.WithError(err).Error("unable to upload group img")
		return
	}
	msg.Append(upImg)
	qqClient.SendGroupMessage(
		groupMessage.GroupCode,
		msg,
	)

	logger.Debugf("successfully handled group message from group chat %s(%d)",
		groupMessage.GroupName,
		groupMessage.GroupCode,
	)
}

func (m *suki) registerCallbacks(b *bot.Bot) {
	b.OnGroupMessage(m.handleGroupMessage)
}
