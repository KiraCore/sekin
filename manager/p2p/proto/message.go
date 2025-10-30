package proto

import (
	"encoding/json"
	"fmt"
)

type MessageType string

const (
	MessageTypeJoinRequest  MessageType = "join_request"
	MessageTypeJoinResponse MessageType = "join_response"
	MessageTypeMetrics      MessageType = "metrics"
)

type Message struct {
	MsgType    MessageType `json:"type"`
	MsgPayload interface{} `json:"payload"`
}

func (m *Message) Type() string {
	return string(m.MsgType)
}

func (m *Message) Payload() interface{} {
	return m.MsgPayload
}

func NewMessage(msgType MessageType, payload interface{}) *Message {
	return &Message{
		MsgType:    msgType,
		MsgPayload: payload,
	}
}

func UnmarshalPayload(payload interface{}, target interface{}) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	if err := json.Unmarshal(bytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return nil
}
