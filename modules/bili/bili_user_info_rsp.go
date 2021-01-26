package bili

const (
	// LiveRoom.LiveStatus
	NotStreaming = 0
	Streaming    = 1
)

type UserInfoRsp struct {
	Code    int
	Message string
	Ttl     int
	Data    UserInfo
}

type UserInfo struct {
	Mid      int
	Name     string
	Sex      string
	Face     string
	Sign     string
	Rank     int
	Level    int
	TopPhoto string   `json:"top_photo"`
	LiveRoom LiveRoom `json:"live_room"`
}

type LiveRoom struct {
	RoomStatus  int
	LiveStatus  int
	Url         string
	Title       string
	Cover       string
	Online      int
	RoomId      int
	RoundStatus int
}
