package naive_chatbot

import "errors"

// These are APIs exposed to other modules.
// They should only be called after initialization of all modules.

func GetTriggerProb(groupId int64) (float32, error) {
	if instance == nil {
		return 0.0, errors.New("naive_chatbot disabled")
	}
	if tp, ok := instance.groupTriggerProb[groupId]; !ok {
		return 0.0, errors.New("invalid groupId")
	} else {
		return tp, nil
	}
}

func SetTriggerProb(groupId int64, newProb float32) error {
	if instance == nil {
		return errors.New("naive_chatbot disabled")
	}
	if newProb < 0.0 || newProb > 1.0 {
		return errors.New("trigger probability out of range [0, 1]")
	}
	if _, ok := instance.groupTriggerProb[groupId]; !ok {
		return errors.New("invalid groupId")
	} else {
		instance.groupTriggerProb[groupId] = newProb
		return nil
	}
}
