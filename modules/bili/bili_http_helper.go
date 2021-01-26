package bili

import (
	"encoding/json"
)
import "errors"
import "net/http"
import "strconv"
import "time"
import "io/ioutil"

const (
	GetUserInfoApi     = "https://api.bilibili.com/x/space/acc/info"
	GetLiveRoomInfoApi = "https://api.live.bilibili.com/room/v1/Room/room_init"

	DefaultTimeout = 5 * time.Second
)

func GetUserInfo(bid int64) (*UserInfo, error) {
	req, err := http.NewRequest("GET", GetUserInfoApi, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("mid", strconv.FormatInt(bid, 10))
	req.URL.RawQuery = q.Encode()

	client := http.Client{
		Timeout: DefaultTimeout,
	}

	rsp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	userInfoRsp := UserInfoRsp{}
	err = json.Unmarshal(body, &userInfoRsp)
	if err != nil {
		return nil, err
	}

	if userInfoRsp.Code != 0 {
		return nil, errors.New("GetUserInfoApi return code is ERROR")
	}
	return &userInfoRsp.Data, nil
}
