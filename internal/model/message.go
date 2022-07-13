package model

import "time"

type SendMessage struct {
	To string `json:"to"`
}

type ReceiveMessage struct {
	From    string     `json:"from"`
	Msg     string     `json:"msg"`
	SendDtm *time.Time `json:"send_dtm"`
}
