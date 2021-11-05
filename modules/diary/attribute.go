package diary

import (
	"fmt"
	"strings"
)

type Attribute struct {
	Ttl   int64 `json:"ttl"`
	TtlTs int64 `json:"ttlTs"`

	DrawingPower   int64 `json:"drawingPower"`
	Health         int64 `json:"health"`
	Accomplishment int64 `json:"accomplishment"`
	Experience     int64 `json:"experience"`
	Friendship     int64 `json:"friendship"`
}

func (attr *Attribute) Add(op *Attribute) *Attribute {
	attr.Ttl += op.Ttl
	attr.TtlTs += op.TtlTs
	attr.DrawingPower += op.DrawingPower
	attr.Health += op.Health
	attr.Accomplishment += op.Accomplishment
	attr.Experience += op.Experience
	attr.Friendship += op.Friendship
	return attr
}

func (attr *Attribute) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("寿命：%d\n", attr.Ttl))
	sb.WriteString(fmt.Sprintf("图力：%d\n", attr.DrawingPower))
	sb.WriteString(fmt.Sprintf("健康：%d\n", attr.Health))
	sb.WriteString(fmt.Sprintf("成就：%d\n", attr.Accomplishment))
	sb.WriteString(fmt.Sprintf("经验：%d\n", attr.Experience))
	sb.WriteString(fmt.Sprintf("友情：%d", attr.Friendship))

	return sb.String()
}
