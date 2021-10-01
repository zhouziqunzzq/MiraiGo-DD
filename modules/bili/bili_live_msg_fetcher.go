package bili

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

// https://github.com/lovelyyoshino/Bilibili-Live-API/blob/master/API.WebSocket.md

const (
	BiliLiveMsgApi    = "wss://broadcastlv.chat.bilibili.com:2245/sub"
	HeartBeatInterval = 30
	MaxReconnection   = 10
	ReadTimeout       = 10 * time.Second
	WriteTimeout      = 10 * time.Second
)

type LiveMsgFetcher struct {
	UserInfo   *UserInfo
	RoomID     int64
	conn       *websocket.Conn
	hbTicker   *time.Ticker
	eventChan  chan<- *Event
	closeChan  chan bool
	quitHbChan chan bool
	quitWg     sync.WaitGroup
	writeMu    sync.Mutex
	isRunning  bool
}

func NewLiveMsgFetcher(userInfo *UserInfo, eventChan chan *Event) *LiveMsgFetcher {
	return &LiveMsgFetcher{
		UserInfo:   userInfo,
		RoomID:     int64(userInfo.LiveRoom.RoomId),
		conn:       nil,
		hbTicker:   time.NewTicker(HeartBeatInterval * time.Second),
		eventChan:  eventChan,
		closeChan:  make(chan bool),
		quitHbChan: make(chan bool),
		isRunning:  false,
	}
}

func (f *LiveMsgFetcher) Init() error {
	// connect to bilibili live message websocket API
	var err error
	f.conn, _, err = websocket.DefaultDialer.Dial(BiliLiveMsgApi, nil)
	if err != nil {
		return err
	}

	// join the live room
	err = f.sendJoinRequest()
	if err != nil {
		return err
	}

	return nil
}

func (f *LiveMsgFetcher) reconnect() bool {
	f.writeMu.Lock()
	defer f.writeMu.Unlock()

	for i := 1; i <= MaxReconnection; i++ {
		logger.Warnf("trying to reconnect to room %d (%d/%d)", f.RoomID, i, MaxReconnection)
		err := f.Init()
		if err == nil {
			return true
		} else {
			logger.WithError(err).Errorf("reconnect failed for room %d", f.RoomID)
		}
	}
	return false
}

func (f *LiveMsgFetcher) Run() {
	if f.isRunning {
		logger.Infof("live msg fetcher for live room %d is already running, ignoring...", f.RoomID)
		return
	} else {
		defer func() { f.isRunning = true }()
	}

	f.writeMu.Lock()
	defer f.writeMu.Unlock()

	// start heartbeat coroutine
	go func() {
		f.quitWg.Add(1)
		defer f.quitWg.Done()
		logger.Infof("heartbeat coroutine for live room %d has started", f.RoomID)
		for {
			err := f.sendHeartBeat()
			if err != nil {
				logger.WithError(err).Errorf("failed to send heartbeat for live room %d", f.RoomID)
				// try to reconnect
				if f.reconnect() {
					continue
				} else {
					logger.Errorf("unable to reconnect for room %d, stopping live msg fetcher", f.RoomID)
					f.closeChan <- true
					return
				}
			}
			select {
			case <-f.hbTicker.C:
				continue
			case <-f.quitHbChan:
				f.hbTicker.Stop()
				logger.Infof("heartbeat coroutine for live room %d has stopped", f.RoomID)
				return
			}
		}
	}()

	// start main message loop coroutine
	go func() {
		f.quitWg.Add(1)
		defer f.quitWg.Done()
		defer f.conn.Close()
		defer func() { f.quitHbChan <- true }()
		logger.Infof("main message loop coroutine for live room %d has started", f.RoomID)

		for {
			select {
			case <-f.closeChan:
				logger.Infof("main message loop coroutine for live room %d has stopped", f.RoomID)
				return
			default:
				_ = f.conn.SetReadDeadline(time.Now().Add(ReadTimeout))
				_, buffer, err := f.conn.ReadMessage()
				if err != nil {
					logger.WithError(err).Errorf("failed to read message for live room %d", f.RoomID)
					// try to reconnect
					if f.reconnect() {
						continue
					} else {
						logger.Errorf("unable to reconnect for room %d, stopping live msg fetcher", f.RoomID)
						return
					}
				}

				msg, err := decodeMsg(buffer)
				if err != nil {
					logger.WithError(err).Errorf("failed to decode message for live room %d", f.RoomID)
					continue
				}

				switch msg.Op {
				case JoinReplyOp:
					logger.Infof("加入房间 %d", f.RoomID)
				case HeartBeatReplyOp:
					if c, ok := msg.Data.(uint32); ok {
						logger.Infof("房间 %d 的人气值为 %d", f.RoomID, c)
					} else {
						logger.Warnf("failed to convert viewer count to uint32")
					}
				case NotificationOp:
					if nList, ok := msg.Data.([]*NotificationBody); ok {
						for _, nb := range nList {
							uname, content, err := nb.ParseAsDanmu()
							if err != nil {
								logger.WithError(err).Errorf("failed to parse notification as danmu")
							} else {
								f.eventChan <- NewEvent(NewDanmu, &DanmuEventData{
									FromUserName:     uname,
									Content:          content,
									StreamerUserInfo: f.UserInfo,
								})
								logger.Infof("房间: %d - %s: %s", f.RoomID, uname, content)
							}
						}
					}
				default:
					logger.Warnf("unhandled msg with op=%d", msg.Op)
				}
			}
		}
	}()
}

func (f *LiveMsgFetcher) Stop() {
	f.closeChan <- true
	f.quitWg.Wait()
	f.isRunning = false
}

func (f *LiveMsgFetcher) sendJoinRequest() error {
	req := JoinRequestBody{
		Platform: "web",
		RoomID:   f.RoomID,
	}
	reqJson, err := json.Marshal(req)
	if err != nil {
		return err
	}

	reqBin := encodeMsg(reqJson, JoinOp, JsonProcVer)
	f.writeMu.Lock()
	defer f.writeMu.Unlock()
	_ = f.conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
	err = f.conn.WriteMessage(websocket.BinaryMessage, reqBin)
	if err != nil {
		return err
	}

	return nil
}

func (f *LiveMsgFetcher) sendHeartBeat() error {
	reqBin := encodeMsg([]byte{}, HeartBeatOp, Uint32ProcVer)
	f.writeMu.Lock()
	defer f.writeMu.Unlock()
	_ = f.conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
	err := f.conn.WriteMessage(websocket.BinaryMessage, reqBin)
	if err != nil {
		return err
	}

	logger.Info("heartbeat sent")
	return nil
}

func encodeMsg(payload []byte, op uint32, procVer uint16) []byte {
	p := LiveMsgPacket{
		PacketLen: 0, // will be auto calculated
		HeaderLen: HeaderLen,
		ProcVer:   procVer,
		Op:        op,
		SeqID:     1,
		Body:      payload,
	}

	return p.ToBytes()
}

func decodeMsg(buffer []byte) (*DecodedLiveMsg, error) {
	p := new(LiveMsgPacket)
	err := p.FromBytes(buffer)
	if err != nil {
		return nil, err
	}

	switch p.Op {
	case JoinReplyOp:
		return &DecodedLiveMsg{
			Op:   p.Op,
			Data: nil,
		}, nil
	case HeartBeatReplyOp:
		c, err := p.DecodeBodyAsViewCnt()
		if err != nil {
			return nil, err
		} else {
			return &DecodedLiveMsg{
				Op:   p.Op,
				Data: c,
			}, nil
		}
	case NotificationOp:
		rawJsonList, err := p.DecodeBodyAsNotificationRawJson()
		if err != nil {
			return nil, err
		}

		parsedBodyList := make([]*NotificationBody, 0)
		for _, rawJson := range rawJsonList {
			var b NotificationBody
			err = json.Unmarshal([]byte(rawJson), &b)
			if err != nil {
				return nil, err
			}
			// we return just danmu for now
			if b.Cmd == NotificationDanmuCmd {
				parsedBodyList = append(parsedBodyList, &b)
			}
		}
		return &DecodedLiveMsg{
			Op:   p.Op,
			Data: parsedBodyList,
		}, nil
	default:
		return &DecodedLiveMsg{
			Op:   p.Op,
			Data: nil,
		}, nil
	}
}
