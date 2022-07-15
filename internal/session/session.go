package session

import (
	"chat-session/internal/cache"
	"chat-session/internal/model"
	"chat-session/internal/repository"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"go.uber.org/zap"
	"net"
	"net/http"
	"time"
)

const (
	cannotConnect = "cannot connect"
)
const (
	rdbOnline      = "%s-online"
	rdbPublish     = "%s-channel"
	rdbUndelivered = "%s-undelivered"
)
const (
	statusOnline  = "online"
	statusOffline = "offline"
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

	//setup status to online
	s.setStatus(ss, statusOnline)

	//get undelivered message while user offline
	s.getUndeliveredMsg(ss)

	//read message from client
	s.readClientMsg(ss)
}

func (s service) getUndeliveredMsg(ss *SsModel) {
	//get update flag from redis first. if key found then it means need to update otherwise do nothing.
	_, err := s.cache.Get(fmt.Sprintf(rdbUndelivered, ss.Username))
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
	var ok []int64
	for _, entity := range entities {
		tmp := model.ChatMessage{
			ReceiverId: entity.ReceiverId, //to whom?
			SenderId:   entity.SenderId,   //from whom?
			Msg:        entity.Message,    //message
			SendDtm:    entity.SendDtm,    //when?
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
		ok = append(ok, entity.Id)
	}

	if len(ok) > 0 {
		//update undelivered message to read when send to client successfully
		err = s.messageRepo.UpdateIsRead(ok)
		if err != nil {
			zap.S().Errorf("s.messageRepo.UpdateIsRead: %v", err)
			return
		}
	}

	//if send success then mark consume flag to
	err = s.cache.Del(ss.Username)
	if err != nil {
		zap.S().Error("s.cache.Del: %v", err)
	}
}

func (s service) readClientMsg(ss *SsModel) {
	var endChan = make(chan bool)
	defer close(endChan)
	go func() {
		for {
			data, _, err := wsutil.ReadClientData(ss.Conn)
			if err != nil {
				switch err.(type) {
				case wsutil.ClosedError:
					//remove online status
					s.setStatus(ss, statusOffline)
				default:
					zap.S().Error("cannot read message from client: %v", err)
				}
				endChan <- true
				_ = ss.Conn.Close()
				break
			}

			go s.forwardMsgToReceiver(data)
		}
	}()

	//subscribe redis channel until connection close
	s.subscribeMsg(ss, endChan)
}

func (s service) forwardMsgToReceiver(data []byte) {
	//check if target user is now online, if yes publish message into redis pub/sub and then insert the msg into db with is_read is one (read)
	//make sure that message delivered to target otherwise system should insert data into database instead
	var reqMsg model.ChatMessage
	err := json.Unmarshal(data, &reqMsg)
	if err != nil {
		zap.S().Errorf("invalid request json format: %v", err)
		return
	}

	//check if target user online
	var r int64
	_, err = s.cache.Get(fmt.Sprintf(rdbOnline, reqMsg.ReceiverId))
	if err == nil {
		//target user is not online then publish message into channel
		to := fmt.Sprintf(rdbPublish, reqMsg.ReceiverId)
		r, err = s.cache.Pub(to, string(data)).Result()
		if err != nil {
			zap.S().Error("s.cache.Pub: %v", err)
			goto offline
		}
		if r == 0 {
			zap.S().Infof("not found %s then insert chat-message into database", to)
			goto offline
		}
		return
	}

	//no cache found
	if err == redis.Nil {
		zap.S().Infof("%s is not online then insert chat-message into database", reqMsg.ReceiverId)
		goto offline
	}

	//err while get cache
	if err != nil {
		zap.S().Error("error while get user status then insert chat-message into database")
		goto offline
	}

offline:
	s.saveMsg(reqMsg, time.Now(), false)

	//set
	err = s.cache.Set(fmt.Sprintf(rdbUndelivered, reqMsg.ReceiverId), time.Now().Format(time.RFC3339), 24*time.Hour)
	if err != nil {
		zap.S().Errorf("s.cache.Set: %v", err)
	}
}

type SsModel struct {
	Conn     net.Conn
	Username string `json:"username"`
}

func initConnection(w http.ResponseWriter, r *http.Request) (*SsModel, error) {
	username := chi.URLParam(r, "username")
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		return nil, err
	}

	return &SsModel{
		Conn:     conn,
		Username: username,
	}, nil
}

func (s service) setStatus(ss *SsModel, status string) {
	//check if user disconnect
	msg := fmt.Sprintf(rdbOnline, ss.Username)

	//turned status to online
	if status == statusOnline {
		zap.S().Infof("%s is now online", ss.Username)
		err := s.cache.Set(msg, msg, 24*time.Hour)
		if err != nil {
			zap.S().Error("s.cache.Set: %v", err)
		}
		return
	}

	//turned status to offline
	if status == statusOffline {
		zap.S().Infof("%s is now offline", ss.Username)
		err := s.cache.Del(msg)
		if err != nil {
			zap.S().Error("s.cache.Del: %v", err)
		}
		return
	}

	zap.S().Error("invalid status")
}

func (s service) subscribeMsg(ss *SsModel, endChan chan bool) {
	run := true
	ps := s.cache.Sub(fmt.Sprintf(rdbPublish, ss.Username))
	for run {
		select {
		case <-endChan:
			zap.S().Infof("%s stop subscribe message", ss.Username)
			run = false
		default:
			msg, err := ps.ReceiveTimeout(context.Background(), time.Second)
			if err != nil {
				switch err.(type) {
				case *net.OpError:
					//timeout here
					continue
				default:
					zap.S().Errorf("s.cache.Sub: %v", err)
					continue
				}
			}
			s.writeServerMessage(ss, msg)
		}
	}
}

func (s service) writeServerMessage(ss *SsModel, msg interface{}) {
	switch msg := msg.(type) {
	case *redis.Message:
		//unmarshal message and
		var m model.ChatMessage
		var n = time.Now()
		err := json.Unmarshal([]byte(msg.Payload), &m)
		if err != nil {
			zap.S().Errorf("json.Unmarshal: %v", err)
			return
		}
		m.SendDtm = &n

		//return message back if success
		j, _ := json.Marshal(&m)
		err = wsutil.WriteServerMessage(ss.Conn, ws.OpText, j)
		if err != nil {
			_ = ss.Conn.Close()
			return
		}
		s.saveMsg(m, n, true)
	default:
		//do nothing
	}
}

func (s service) saveMsg(m model.ChatMessage, n time.Time, isRead bool) error {
	e := repository.MessageEntity{
		ReceiverId: m.ReceiverId,
		SenderId:   m.SenderId,
		Message:    m.Msg,
		IsRead:     isRead,
		SendDtm:    &n,
	}
	if isRead {
		e.ReadDtm = &n
	}
	err := s.messageRepo.Create(e)
	if err != nil {
		zap.S().Errorf("s.messageRepo.Create: %v", err)
	}
	return err
}
