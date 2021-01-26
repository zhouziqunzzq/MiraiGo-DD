package bili

import (
	"fmt"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/config"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
	"gopkg.in/yaml.v2"
	"sync"
	"time"
)

type bili struct {
	isEnabled            bool
	config               Config
	groupIdToBiliUidList map[int64][]int64 // subscriptionRwMu protected
	biliUidToGroupIdList map[int64][]int64 // subscriptionRwMu protected
	subscriptionRwMu     sync.RWMutex
	biliUserInfoBuf      map[int64]*UserInfo // infoBufRwMu protected
	infoBufRwMu          sync.RWMutex
	eventChan            chan *Event
	quitPolling          chan bool
	quitBroadcasting     chan bool
}

func NewBili() *bili {
	return &bili{
		isEnabled:            false,
		config:               Config{},
		groupIdToBiliUidList: make(map[int64][]int64),
		biliUidToGroupIdList: make(map[int64][]int64),
		biliUserInfoBuf:      make(map[int64]*UserInfo),
		eventChan:            make(chan *Event),
		quitPolling:          make(chan bool),
		quitBroadcasting:     make(chan bool),
	}
}

func (m *bili) MiraiGoModule() bot.ModuleInfo {
	return bot.ModuleInfo{
		ID:       ModuleName,
		Instance: instance,
	}
}

func (m *bili) Init() {
	// check is_enabled
	m.isEnabled = config.GlobalConfig.GetBool("modules." + ModuleName + ".is_enabled")
	if !m.isEnabled {
		logger.Info("this module is disabled by global config")
		return
	}

	// load module config
	configPath := config.GlobalConfig.GetString("modules." + ModuleName + ".config_path")
	if configPath == "" {
		configPath = "./bili.yaml"
	}
	logger.Debugf("reading config from %s", configPath)
	cb := utils.ReadFile(configPath)
	err := yaml.Unmarshal(cb, &m.config)
	if err != nil {
		logger.WithError(err).Errorf("unable to read config file in %s", configPath)
		m.isEnabled = false
		return
	}

	// load subscription
	// group ID -> bilibili UID list
	m.groupIdToBiliUidList = m.config.Subscription
	logger.Infof("group ID to bilibili UID List: %v", m.groupIdToBiliUidList)
	// bilibili UID -> group ID list (inverse mapping)
	for groupId, BiliUidList := range m.groupIdToBiliUidList {
		for _, uid := range BiliUidList {
			if _, ok := m.biliUidToGroupIdList[uid]; !ok {
				m.biliUidToGroupIdList[uid] = make([]int64, 0)
			}
			m.biliUidToGroupIdList[uid] = append(m.biliUidToGroupIdList[uid], groupId)
		}
	}
	logger.Infof("bilibili UID to group ID List: %v", m.biliUidToGroupIdList)
}

func (m *bili) PostInit() {
	// 第二次初始化
	// 再次过程中可以进行跨Module的动作
	// 如通用数据库等等
}

func (m *bili) Serve(b *bot.Bot) {
	if m.isEnabled {
		m.registerCallbacks(b)
	}
}

func (m *bili) Start(b *bot.Bot) {
	// 此函数会新开携程进行调用
	// ```go
	// 		go exampleModule.Start()
	// ```

	// 可以利用此部分进行后台操作
	// 如http服务器等等

	// start polling coroutine
	go func() {
		ticker := time.NewTicker(time.Duration(m.config.PollingInterval) * time.Second)
		for {
			// perform polling at the very beginning
			m.pollBiliUserInfo()
			select {
			case <-ticker.C:
				continue
			case <-m.quitPolling:
				ticker.Stop()
				return
			}
		}
	}()

	// start event broadcasting coroutine
	go func() {
		for {
			select {
			case e := <-m.eventChan:
				switch e.Type {
				case StartLive:
					if userInfo, ok := e.Data.(*UserInfo); ok {
						m.broadcastStartLiveMsg(b.QQClient, userInfo)
					} else {
						logger.Errorf("unknown event data provided for StartLive, event: %v", e)
					}
				case StopLive:
					if userInfo, ok := e.Data.(*UserInfo); ok {
						m.broadcastStopLiveMsg(b.QQClient, userInfo)
					} else {
						logger.Errorf("unknown event data provided for StopLive, event: %v", e)
					}
				default:
					logger.Debugf("unknown event type %d encountered, skipping", e.Type)
				}
			case <-m.quitBroadcasting:
				return
			}
		}
	}()
}

func (m *bili) Stop(b *bot.Bot, wg *sync.WaitGroup) {
	// 别忘了解锁
	defer wg.Done()
	// 结束部分
	// 一般调用此函数时，程序接收到 os.Interrupt 信号
	// 即将退出
	// 在此处应该释放相应的资源或者对状态进行保存

	// stop polling coroutine
	m.quitPolling <- true

	// stop broadcasting coroutine
	m.quitBroadcasting <- true
}

func (m *bili) registerCallbacks(b *bot.Bot) {
	//b.OnGroupMessage(m.handleGroupMessage)
}

func (m *bili) getBiliUidList() []int64 {
	m.subscriptionRwMu.RLock()
	defer m.subscriptionRwMu.RUnlock()

	l := make([]int64, 0, len(m.biliUidToGroupIdList))
	for uid, _ := range m.biliUidToGroupIdList {
		l = append(l, uid)
	}

	return l
}

func (m *bili) pollBiliUserInfo() {
	logger.Info("start polling subscribed bilibili user info")

	uidList := m.getBiliUidList()
	for _, uid := range uidList {
		// call http api
		newUserInfo, err := GetUserInfo(uid)
		if err != nil {
			logger.WithError(err).Errorf("failed to get user info for bid=%d", uid)
			continue
		}

		m.infoBufRwMu.Lock()

		// update info buf and trigger events
		if oldUserInfo, ok := m.biliUserInfoBuf[uid]; ok {
			if oldUserInfo.LiveRoom.LiveStatus == NotStreaming && newUserInfo.LiveRoom.LiveStatus == Streaming {
				logger.Infof("bilibili user %s(%d) has started streaming", newUserInfo.Name, uid)
				e := NewEvent(StartLive, newUserInfo)
				m.eventChan <- e
			} else if oldUserInfo.LiveRoom.LiveStatus == Streaming && newUserInfo.LiveRoom.LiveStatus == NotStreaming {
				logger.Infof("bilibili user %s(%d) has stopped streaming", newUserInfo.Name, uid)
				e := NewEvent(StopLive, newUserInfo)
				m.eventChan <- e
			}
		}
		m.biliUserInfoBuf[uid] = newUserInfo

		m.infoBufRwMu.Unlock()
	}

	logger.Infof("finish polling %d subscribed bilibili user info", len(uidList))
}

func (m *bili) broadcastStartLiveMsg(qqClient *client.QQClient, userInfo *UserInfo) {
	msg := message.NewSendingMessage()
	msg.Append(message.NewText(fmt.Sprintf(
		"您关注的%s开播啦！快去直播间 DD 吧～\n直播间标题：%s\n直播间链接：%s",
		userInfo.Name, userInfo.LiveRoom.Title, userInfo.LiveRoom.Url,
	)))

	m.broadcastMsgToSubscribedGroup(qqClient, msg, int64(userInfo.Mid))
}

func (m *bili) broadcastStopLiveMsg(qqClient *client.QQClient, userInfo *UserInfo) {
	msg := message.NewSendingMessage()
	msg.Append(message.NewText(fmt.Sprintf(
		"您关注的%s下播啦！感谢观看，记得下次再来 DD 哦～",
		userInfo.Name,
	)))

	m.broadcastMsgToSubscribedGroup(qqClient, msg, int64(userInfo.Mid))
}

func (m *bili) broadcastMsgToSubscribedGroup(qqClient *client.QQClient, msg *message.SendingMessage, bid int64) {
	m.subscriptionRwMu.RLock()
	defer m.subscriptionRwMu.RUnlock()

	if l, ok := m.biliUidToGroupIdList[bid]; !ok {
		logger.Errorf("bilibili user %d not found in subscription map", bid)
		return
	} else {
		for _, groupId := range l {
			qqClient.SendGroupMessage(groupId, msg)
		}
	}
}
