package mock

import (
	"chat-session/internal/repository"
	"chat-session/internal/tests/mock_repository"
	"errors"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

func ChatMessageRepo(t *testing.T, tc string, n time.Time) *mock_repository.MockMessage {
	switch tc {
	case "OK":
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockMessage(mockCtrl)
		repo.EXPECT().Create(repository.MessageEntity{SendDtm: &n}).Return(nil).AnyTimes()
		repo.EXPECT().FindNewMsgByReceiverId("uefa").Return([]repository.MessageEntity{}, nil).AnyTimes()
		repo.EXPECT().UpdateIsRead([]int64{1}).Return(nil).AnyTimes()
		return repo
	case "!OK":
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockMessage(mockCtrl)
		repo.EXPECT().Create(repository.MessageEntity{SendDtm: &n}).Return(errors.New("mock err")).AnyTimes()
		repo.EXPECT().FindNewMsgByReceiverId("uefa").Return(nil, errors.New("mock err")).AnyTimes()
		repo.EXPECT().UpdateIsRead([]int64{1}).Return(errors.New("mock err")).AnyTimes()
		return repo
	}

	panic("no case found")
}
