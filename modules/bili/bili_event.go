package bili

const (
	StartLive int = iota
	StopLive
)

type Event struct {
	Type int
	Data interface{}
}

func NewEvent(eventType int, eventData interface{}) *Event {
	return &Event{
		Type: eventType,
		Data: eventData,
	}
}
