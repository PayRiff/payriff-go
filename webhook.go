package payriff

import (
	"bytes"
	"encoding/json"
)

var errNotACallback = invalid("not a Payriff order callback")

func ParseOrderCallback(body []byte) (*OrderInfo, error) {
	var root struct {
		Payload json.RawMessage `json:"payload"`
	}
	if json.Unmarshal(body, &root) != nil {
		return nil, errNotACallback
	}
	payload := bytes.TrimSpace(root.Payload)
	if len(payload) == 0 || payload[0] != '{' {
		return nil, errNotACallback
	}
	var probe struct {
		OrderID *string `json:"orderId"`
	}
	if json.Unmarshal(payload, &probe) != nil || probe.OrderID == nil {
		return nil, errNotACallback
	}
	var order OrderInfo
	if err := json.Unmarshal(payload, &order); err != nil {
		return nil, errNotACallback
	}
	return &order, nil
}
