package wms

import (
	"encoding/json"
)

func encodeMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

func decodeMessage(data []byte) (Message, error) {
	var msg Message

	if err := json.Unmarshal(data, &msg); err != nil {
		return Message{}, err
	}

	return msg, nil
}
