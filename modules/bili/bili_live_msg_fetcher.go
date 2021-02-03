package bili

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"time"
)

// https://github.com/lovelyyoshino/Bilibili-Live-API/blob/master/API.WebSocket.md

const (
	BiliLiveMsgApi    = "wss://broadcastlv.chat.bilibili.com:2245/sub"
	HeartBeatInterval = 30
)

type LiveMsgFetcher struct {
	UserInfo   *UserInfo
	RoomID     int64
	conn       *websocket.Conn
	hbTicker   *time.Ticker
	eventChan  chan<- *Event
	closeChan  chan bool
	quitHbChan chan bool
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

func (f *LiveMsgFetcher) Run() {
	// start heartbeat coroutine
	go func() {
		for {
			err := f.sendHeartBeat()
			if err != nil {
				logger.WithError(err).Errorf("failed to send heartbeat for live room %d", f.RoomID)
			}
			select {
			case <-f.hbTicker.C:
				continue
			case <-f.quitHbChan:
				f.hbTicker.Stop()
				return
			}
		}
	}()

	// start main message loop coroutine
	go func() {
		defer f.conn.Close()
		defer func() {
			f.quitHbChan <- true
		}()

		for {
			select {
			case <-f.closeChan:
				return
			default:
				_, buffer, err := f.conn.ReadMessage()
				if err != nil {
					logger.WithError(err).Errorf("failed to read message for live room %d", f.RoomID)
				}

				msg, err := decodeMsg(buffer)
				if err != nil {
					logger.WithError(err).Errorf("failed to decode message for live room %d", f.RoomID)
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
	err = f.conn.WriteMessage(websocket.BinaryMessage, reqBin)
	if err != nil {
		return err
	}

	return nil
}

func (f *LiveMsgFetcher) sendHeartBeat() error {
	reqBin := encodeMsg([]byte{}, HeartBeatOp, Uint32ProcVer)
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
