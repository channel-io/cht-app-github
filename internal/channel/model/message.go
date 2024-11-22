package model

type Message struct {
	Blocks []MessageBlock `json:"blocks"`
}

func NewMessage(blocks ...MessageBlock) *Message {
	return &Message{
		Blocks: blocks,
	}
}
