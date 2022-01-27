package bili

import (
	"fmt"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/zhouziqunzzq/MiraiGo-DD/bot"
	"github.com/zhouziqunzzq/MiraiGo-DD/config"
	"github.com/zhouziqunzzq/MiraiGo-DD/utils"
	"gopkg.in/yaml.v2"
	"strings"
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
	biliUidToMsgFetcher  map[int64]*LiveMsgFetcher // fetcherRwMu protected
	fetcherRwMu          sync.RWMutex
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
		biliUidToMsgFetcher:  make(map[int64]*LiveMsgFetcher),
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
		// wait until bot is online
		for {
			if b.Online.Load() {
				break
			} else {
				time.Sleep(1 * time.Second)
			}
		}

		logger.Info("starting event broadcasting coroutine")
		for {
			select {
			case e := <-m.eventChan:
				switch e.Type {
				case StartLive:
					if userInfo, ok := e.Data.(*UserInfo); ok {
						m.broadcastStartLiveMsg(b.QQClient, userInfo)
						// start fetching danmu for this user
						m.runLiveMsgFetcherForBiliUser(int64(userInfo.Mid))
					} else {
						logger.Errorf("unknown event data provided for StartLive, event: %v", e)
					}
				case StopLive:
					if userInfo, ok := e.Data.(*UserInfo); ok {
						m.broadcastStopLiveMsg(b.QQClient, userInfo)
						// stop fetching danmu for this user
						// Note: BLOCKING call! Call from a new goroutine!
						go m.stopLiveMsgFetcherForBiliUser(int64(userInfo.Mid))
					} else {
						logger.Errorf("unknown event data provided for StopLive, event: %v", e)
					}
				case NewDanmu:
					if danmuData, ok := e.Data.(*DanmuEventData); ok {
						// check danmu keywords
						for _, w := range m.config.DanmuForwardKeywords {
							if strings.Contains(danmuData.Content, w) {
								m.broadcastDanmu(b.QQClient, danmuData)
								break
							}
						}
					} else {
						logger.Errorf("unknown event data provided for NewDanmu, event: %v", e)
					}
				default:
					logger.Debugf("unknown event type %d encountered, skipping", e.Type)
				}
			case <-m.quitBroadcasting:
				return
			}
		}
	}()

	// test live msg fetcher
	//m.infoBufRwMu.Lock()
	//m.biliUserInfoBuf[407106379] = &UserInfo{
	//	Mid:  407106379,
	//	Name: "test",
	//	LiveRoom: LiveRoom{
	//		RoomStatus: Streaming,
	//		RoomId:     21396545,
	//	},
	//}
	//m.infoBufRwMu.Unlock()
	//m.runLiveMsgFetcherForBiliUser(407106379)
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

	// stop live fetchers
	m.fetcherRwMu.RLock()
	for _, f := range m.biliUidToMsgFetcher {
		f.Stop()
	}
	m.fetcherRwMu.RUnlock()
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
	logger.Debug("start polling subscribed bilibili user info")

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
			// we have old user info, check status change
			if oldUserInfo.LiveRoom.LiveStatus == NotStreaming && newUserInfo.LiveRoom.LiveStatus == Streaming {
				logger.Infof("bilibili user %s(%d) has started streaming", newUserInfo.Name, uid)
				e := NewEvent(StartLive, newUserInfo)
				m.eventChan <- e
			} else if oldUserInfo.LiveRoom.LiveStatus == Streaming && newUserInfo.LiveRoom.LiveStatus == NotStreaming {
				logger.Infof("bilibili user %s(%d) has stopped streaming", newUserInfo.Name, uid)
				e := NewEvent(StopLive, newUserInfo)
				m.eventChan <- e
			}
		} else {
			// we don't have old user info, just check current status
			if newUserInfo.LiveRoom.LiveStatus == Streaming {
				logger.Infof("bilibili user %s(%d) has started streaming", newUserInfo.Name, uid)
				e := NewEvent(StartLive, newUserInfo)
				m.eventChan <- e
			}
		}
		m.biliUserInfoBuf[uid] = newUserInfo

		m.infoBufRwMu.Unlock()
	}

	logger.Debugf("finish polling %d subscribed bilibili user info", len(uidList))
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

func (m *bili) broadcastDanmu(qqClient *client.QQClient, danmuData *DanmuEventData) {
	userInfo := danmuData.StreamerUserInfo
	msg := message.NewSendingMessage()
	msg.Append(message.NewText(fmt.Sprintf(
		"【弹幕中继】\n主播：%s\n直播间标题：%s\n发送人：%s\n内容：%s",
		userInfo.Name, userInfo.LiveRoom.Title, danmuData.FromUserName, danmuData.Content,
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

func (m *bili) runLiveMsgFetcherForBiliUser(bid int64) {
	m.fetcherRwMu.Lock()
	defer m.fetcherRwMu.Unlock()

	if fetcher, ok := m.biliUidToMsgFetcher[bid]; ok {
		// Note: we stop the stale live msg fetcher first because the live might
		// have been cut off abnormally and the live msg fetcher had been corrupted.
		logger.Warnf("live msg fetcher instance for bid %d already exist, stopping stable instance...", bid)
		fetcher.Stop()
		delete(m.biliUidToMsgFetcher, bid)
		logger.Infof("successfully stopped live msg fetcher for bid %d", bid)
	}

	// get userinfo from buf
	m.infoBufRwMu.RLock()
	defer m.infoBufRwMu.RUnlock()
	if info, ok := m.biliUserInfoBuf[bid]; !ok {
		logger.Errorf("invalid bili uid %d, ignoring...", bid)
	} else {
		logger.Infof("starting live msg fetcher for bid %d", bid)
		fetcher := NewLiveMsgFetcher(info, m.eventChan)
		initSuccess := false
		for i := 1; i <= MaxReconnection; i++ {
			err := fetcher.Init()
			if err != nil {
				logger.WithError(err).Errorf(
					"failed to initialize live msg fetcher for bid %d, retrying (%d/%d)",
					bid, i, MaxReconnection,
				)
			} else {
				initSuccess = true
				break
			}
		}
		if initSuccess {
			fetcher.Run()
			m.biliUidToMsgFetcher[bid] = fetcher
			logger.Infof("successfully started live msg fetcher for bid %d", bid)
		} else {
			logger.Errorf(
				"failed to initialize live msg fetcher for bid %d after %d attempts",
				bid, MaxReconnection,
			)
		}
	}
}

// Note: BLOCKING call!! Refrain from calling directly from event broadcasting goroutine
// since it might cause deadlock between that goroutine and message loop goroutine.
func (m *bili) stopLiveMsgFetcherForBiliUser(bid int64) {
	m.fetcherRwMu.Lock()
	defer m.fetcherRwMu.Unlock()

	if fetcher, ok := m.biliUidToMsgFetcher[bid]; !ok {
		logger.Warnf("no live msg fetcher instance found for bid %d, ignoring...", bid)
	} else {
		logger.Infof("stopping live msg fetcher for bid %d", bid)
		fetcher.Stop()
		delete(m.biliUidToMsgFetcher, bid)
		logger.Infof("successfully stopped live msg fetcher for bid %d", bid)
	}
}
