package session

import (
	"chat-session/internal/cache"
	"chat-session/internal/model"
	"chat-session/internal/repository"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"go.uber.org/zap"
	"net"
	"net/http"
)

const (
	cannotConnect = "cannot connect"
)

type Service interface {
	Online(w http.ResponseWriter, r *http.Request)
}

type service struct {
	cache       cache.Cache
	messageRepo repository.Message
}

func NewService(cache cache.Cache, messageRepo repository.Message) Service {
	return &service{
		cache:       cache,
		messageRepo: messageRepo,
	}
}

func (s service) Online(w http.ResponseWriter, r *http.Request) {
	ss, err := initConnection(w, r)
	if err != nil {
		http.Error(w, cannotConnect, http.StatusInternalServerError)
		return
	}

	//update missed message
	s.updateMsg(ss)
	s.readMessage(ss)
}

func (s service) updateMsg(ss *SsModel) {
	//get update flag from redis first. if key found then it means need to update otherwise do nothing.
	_, err := s.cache.Get(ss.Username)
	if err == redis.Nil {
		//no need no new message
		return
	}
	if err != nil {
		zap.S().Error("s.cache.Get: %v", err)
		return
	}

	//fetch data from database where consume flg is not mark
	entities, err := s.messageRepo.FindNewMsgByReceiverId(ss.Username)
	if err != nil {
		zap.S().Errorf("s.messageRepo.FindNewMsgByReceiverId: %v", err)
		return
	}

	//send all message to client
	for _, entity := range entities {
		tmp := model.ReceiveMessage{
			From:    entity.SenderId, //from whom?
			Msg:     entity.Message,  //message
			SendDtm: entity.SendDtm,  //when?
		}
		j, err := json.Marshal(&tmp)
		if err != nil {
			zap.S().Error("json.Marshal: %v", err)
			break
		}

		//send message to client
		err = wsutil.WriteServerMessage(ss.Conn, ws.OpText, j)
		if err != nil {
			zap.S().Error("wsutil.WriteServerMessage: %v", err)
			break
		}
	}

	//if send success then mark consume flag to
	err = s.cache.Del(ss.Username)
	if err != nil {
		zap.S().Error("s.cache.Del: %v", err)
	}
}

func (s service) readMessage(ss *SsModel) {
	for {
		data, _, err := wsutil.ReadClientData(ss.Conn)
		if err != nil {
			switch err.(type) {
			case wsutil.ClosedError:
				//check if user disconnect
				zap.S().Infof("%s is offline", ss.Username)
			default:
				zap.S().Error("cannot read message from client: %v", err)
			}
			_ = ss.Conn.Close()
			break
		}

		//return message back if success
		err = wsutil.WriteServerMessage(ss.Conn, ws.OpText, data)
		if err != nil {
			_ = ss.Conn.Close()
			return
		}
		fmt.Printf("%s\n", data)
	}
}

type SsModel struct {
	Conn     net.Conn
	Op       *ws.OpCode
	Username string `json:"username"`
}

func initConnection(w http.ResponseWriter, r *http.Request) (*SsModel, error) {
	username := chi.URLParam(r, "username")
	zap.S().Infof("%s is now online", username)
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		return nil, err
	}

	return &SsModel{
		Conn:     conn,
		Username: username,
	}, nil
}
