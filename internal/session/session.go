package session

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
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
}

func NewService() Service {
	return &service{}
}

type Message struct {
	To string `json:"to"`
}

func (s service) Online(w http.ResponseWriter, r *http.Request) {
	ss, err := initConnection(w, r)
	if err != nil {
		http.Error(w, cannotConnect, http.StatusInternalServerError)
		return
	}

	go s.updateMsg()
	readMessage(ss)
}

func (s service) updateMsg() {
	//get update flag from redis first. if key found then it means need to update otherwise do nothing.
	//fetch data from database where consume flg is not mark
	//send all message to client
	//if send success then mark consume flag to
}

func readMessage(ss *SsModel) {
	for {
		data, _, err := wsutil.ReadClientData(ss.Conn)
		if err != nil {
			fmt.Println(err)
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
	fmt.Println(username)
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		return nil, err
	}

	return &SsModel{
		Conn:     conn,
		Username: username,
	}, nil
}
