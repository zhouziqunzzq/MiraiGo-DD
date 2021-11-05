package diary

import (
	"fmt"
	"time"
)

// These are APIs exposed to other modules.
// They should only be called after initialization of all modules.

func InitDiary(groupId, userId, ttl int64) error {
	id := GetId(groupId, userId)
	attr := Attribute{
		Ttl:   ttl,
		TtlTs: time.Now().Unix(),
		// other attributes = zero
	}

	instance.attributesRwMu.Lock()
	instance.attributes[id] = &attr
	instance.attributesRwMu.Unlock()

	if err := instance.syncAttribute(id); err != nil {
		logger.WithError(err).
			Errorf("failed to sync Attribute with id=%s back to redis", id)
	}

	return nil
}

func QueryDiary(groupId, userId int64) string {
	id := GetId(groupId, userId)
	instance.attributesRwMu.RLock()
	defer instance.attributesRwMu.RUnlock()

	if attr, ok := instance.attributes[id]; ok {
		return fmt.Sprintf("当前属性：\n%s", attr.String())
	} else {
		return "该用户在该群组的日记为空，请初始化后再试"
	}
}

func ApplyEventToDiary(groupId, userId int64, eventId string) string {
	e, ok := Events[eventId]
	if !ok {
		return "事件名称不正确，请检查后再试\n" + ListEvents()
	}

	id := GetId(groupId, userId)
	instance.attributesRwMu.RLock()
	defer instance.attributesRwMu.RUnlock()

	if attr, ok := instance.attributes[id]; ok {
		instance.attributesRwMu.RUnlock()

		instance.attributesRwMu.Lock()
		attr.Add(&e)
		instance.attributesRwMu.Unlock()

		if err := instance.syncAttribute(id); err != nil {
			logger.WithError(err).
				Errorf("failed to sync Attribute with id=%s back to redis", id)
		}

		instance.attributesRwMu.RLock()

		return fmt.Sprintf("事件记录成功！当前属性：\n%s", attr.String())
	} else {
		return "该用户在该群组的日记为空，请初始化之后尝试"
	}
}

func ListEvents() string {
	return "事件列表：\n" +
		"workout: 运动一次\n" +
		"travel: 旅行一次\n" +
		"sketch: 完成练习一次\n" +
		"paint: 完成作品一张\n" +
		"video: 投稿一个视频\n" +
		"chat: 和朋友建立一次深度聊天"
}
