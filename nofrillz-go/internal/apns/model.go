package apns

import "time"

const maxTokenLength = 255

type DeviceToken struct {
	Token     string
	UserID    uint64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DeliveryResult struct {
	StatusCode   int
	Reason       string
	InvalidToken bool
}

type SendSummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Invalid int `json:"invalid"`
	Failed  int `json:"failed"`
}
