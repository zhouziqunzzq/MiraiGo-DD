package bili

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"regexp"
)

const (
	HeaderLen  = 16
	MinJsonLen = 4

	HeartBeatOp      uint32 = 2
	HeartBeatReplyOp uint32 = 3
	NotificationOp   uint32 = 5 // danmaku, broadcast, gift, etc.
	JoinOp           uint32 = 7
	JoinReplyOp      uint32 = 8

	JsonProcVer   uint16 = 0
	Uint32ProcVer uint16 = 1
	ZippedProcVer uint16 = 2

	NotificationDanmuCmd string = "DANMU_MSG"
)

var jsonSplitRegexp = regexp.MustCompile(`[\x00-\x1f]+`)

// ======== WebSocket Packet types ========

type LiveMsgPacket struct {
	PacketLen uint32
	HeaderLen uint16
	ProcVer   uint16
	Op        uint32
	SeqID     uint32
	Body      []byte
}

func (p *LiveMsgPacket) ToBytes() []byte {
	msg := make([]byte, p.HeaderLen)
	p.PacketLen = uint32(p.HeaderLen) + uint32(len(p.Body))

	// Header
	// Packet Length (4)
	binary.BigEndian.PutUint32(msg[0:], p.PacketLen)
	// Header Length (2)
	binary.BigEndian.PutUint16(msg[4:], p.HeaderLen)
	// Protocol Version (2)
	binary.BigEndian.PutUint16(msg[6:], p.ProcVer)
	// Operation (4)
	binary.BigEndian.PutUint32(msg[8:], p.Op)
	// Sequence Id (4)
	binary.BigEndian.PutUint32(msg[12:], p.SeqID)

	// Body
	msg = append(msg, p.Body...)

	return msg
}

func (p *LiveMsgPacket) FromBytes(buffer []byte) error {
	if len(buffer) < HeaderLen {
		return errors.New("corrupted packet")
	}

	// Header
	// Packet Length (4)
	p.PacketLen = binary.BigEndian.Uint32(buffer[0:])
	// Header Length (2)
	p.HeaderLen = binary.BigEndian.Uint16(buffer[4:])
	// Protocol Version (2)
	p.ProcVer = binary.BigEndian.Uint16(buffer[6:])
	// Operation (4)
	p.Op = binary.BigEndian.Uint32(buffer[8:])
	// Sequence Id (4)
	p.SeqID = binary.BigEndian.Uint32(buffer[12:])

	// Body
	p.Body = make([]byte, 0, p.PacketLen-uint32(p.HeaderLen))
	p.Body = append(p.Body, buffer[p.HeaderLen:]...)

	return nil
}

func (p *LiveMsgPacket) DecodeBodyAsViewCnt() (uint32, error) {
	if p.Op != HeartBeatReplyOp {
		return 0, errors.New("op not match")
	}

	if len(p.Body) < 4 {
		return 0, errors.New("corrupted view count body data")
	}

	return binary.BigEndian.Uint32(p.Body), nil
}

func (p *LiveMsgPacket) DecodeBodyAsNotificationRawJson() ([]string, error) {
	if p.Op != NotificationOp {
		return nil, errors.New("op not match")
	}

	if len(p.Body) < 0 {
		return nil, errors.New("corrupted notification body data")
	}

	// unzip the message if necessary
	var jsonToSplit string
	if p.ProcVer == JsonProcVer {
		jsonToSplit = string(p.Body)
	} else if p.ProcVer == ZippedProcVer {
		b := bytes.NewReader(p.Body)
		decodedBuffer := new(bytes.Buffer)
		r, err := zlib.NewReader(b)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(decodedBuffer, r)
		if err != nil {
			return nil, err
		}

		jsonToSplit = decodedBuffer.String()
	} else {
		return nil, errors.New("invalid protocol version for notification body data")
	}

	// split raw json and filter out splitters
	jsonSplit := jsonSplitRegexp.Split(jsonToSplit, -1)
	rst := make([]string, 0, len(jsonSplit))
	for _, j := range jsonSplit {
		if len(j) >= MinJsonLen {
			rst = append(rst, j)
		}
	}

	return rst, nil
}

// ======== Decoded Packet types ========

type DecodedLiveMsg struct {
	Op   uint32
	Data interface{}
}

// ======== Packet Body types ========

type JoinRequestBody struct {
	ClientVer string `json:"clientver,omitempty"`
	Platform  string `json:"platform,omitempty"`
	ProtoVer  int    `json:"protover,omitempty"`
	RoomID    int64  `json:"roomid"`
	UID       int64  `json:"uid,omitempty"`
	Type      int    `json:"type,omitempty"`
}

type NotificationBody struct {
	Cmd  string
	Data json.RawMessage
	Info []json.RawMessage // for DANMU_MSG
}

func (b *NotificationBody) ParseAsDanmu() (uname, content string, err error) {
	if b.Cmd != NotificationDanmuCmd || b.Info == nil || len(b.Info) < 3 {
		err = errors.New("not a danmu notification")
		return
	}

	var uInfo []json.RawMessage
	err = json.Unmarshal(b.Info[2], &uInfo)
	if err != nil {
		return
	}
	err = json.Unmarshal(uInfo[1], &uname)
	if err != nil {
		return
	}

	err = json.Unmarshal(b.Info[1], &content)
	return
}
