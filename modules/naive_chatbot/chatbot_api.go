package naive_chatbot

import "errors"

// These are APIs exposed to other modules.
// They should only be called after initialization of all modules.

func GetTriggerProb() (float32, error) {
	if instance == nil {
		return 0.0, errors.New("naive_chatbot disabled")
	}
	return instance.config.TriggerProb, nil
}

func SetTriggerProb(newProb float32) error {
	if instance == nil {
		return errors.New("naive_chatbot disabled")
	}
	if newProb < 0.0 || newProb > 1.0 {
		return errors.New("trigger probability out of range [0, 1]")
	}
	instance.config.TriggerProb = newProb
	return nil
}
