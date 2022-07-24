package bot

import "github.com/Mrs4s/MiraiGo/client"

type protocol int

const (
	AndroidPhone = protocol(client.AndroidPhone)
	AndroidWatch = protocol(client.AndroidWatch)
	MacOS        = protocol(client.MacOS)
	QiDian       = protocol(client.QiDian)
	IPad         = protocol(client.IPad)
)

// UseProtocol 使用协议
// 不同协议会有部分功能无法使用
// 默认为 AndroidPad
func UseProtocol(p protocol) {
	client.SystemDeviceInfo.Protocol = client.ClientProtocol(p)
}
