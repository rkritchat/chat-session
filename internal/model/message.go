package model

import "time"

type ChatMessage struct {
	SenderId   string     `json:"senderId"`
	ReceiverId string     `json:"receiverId"`
	Msg        string     `json:"msg"`
	SendDtm    *time.Time `json:"send_dtm"`
}
