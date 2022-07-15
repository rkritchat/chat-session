package session

import (
	"chat-session/internal/model"
	"chat-session/internal/repository"
	"chat-session/internal/tests/mock"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_saveMsg(t *testing.T) {
	n := time.Now()
	tt := []struct {
		name            string
		m               model.ChatMessage
		chatMessageRepo repository.Message
		isRead          bool
		expectedE       error
	}{
		{
			name:            "should return nil when create chat message successfully",
			m:               model.ChatMessage{},
			chatMessageRepo: mock.ChatMessageRepo(t, "OK", n),
			isRead:          false,
			expectedE:       nil,
		},
		{
			name:            "should return error when err while create chat message",
			m:               model.ChatMessage{},
			chatMessageRepo: mock.ChatMessageRepo(t, "!OK", n),
			isRead:          false,
			expectedE:       errors.New("mock err"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s := service{messageRepo: tc.chatMessageRepo}
			e := s.saveMsg(tc.m, n, tc.isRead)
			assert.Equal(t, tc.expectedE, e)
		})
	}
}
