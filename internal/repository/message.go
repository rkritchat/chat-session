package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type MessageEntity struct {
	Id         int64      `json:"id"`
	ReceiverId string     `json:"receiver_id"`
	SenderId   string     `json:"sender_id"`
	Message    string     `json:"msg"`
	IsRead     bool       `json:"is_read"`
	SendDtm    *time.Time `json:"send_dtm"`
	ReadDtm    *time.Time `json:"read_dtm"`
}

type Message interface {
	Create(entity MessageEntity) error
	FindNewMsgByReceiverId(receiverId string) ([]MessageEntity, error)
}

type message struct {
	db        *sql.DB
	tableName string
}

func NewMessage(db *sql.DB) Message {
	repo := &message{
		db:        db,
		tableName: "chat_message",
	}
	repo.initTable()
	return repo
}

func (repo message) Create(entity MessageEntity) error {
	stmt, err := repo.db.Prepare(fmt.Sprintf("INSERT INTO %s (receiver_id, sender_id, msg, is_read, send_dtm, read_dtm", repo.tableName))
	if err != nil {
		return err
	}
	_, err = stmt.Exec(entity.ReceiverId, entity.SenderId, entity.Message, entity.IsRead, entity.SendDtm, entity.ReadDtm)
	return err
}

func (repo message) FindNewMsgByReceiverId(receiverId string) ([]MessageEntity, error) {
	stmt, err := repo.db.Prepare(fmt.Sprintf("SELECT receiver_id, sender_id, msg, send_dtm FROM %s WHERE receiver_id = ? AND is_read = 0", repo.tableName))
	if err != nil {
		return nil, err
	}

	r, err := stmt.Query(receiverId)
	if err != nil {
		return nil, err
	}

	var entities []MessageEntity
	for r.Next() {
		var tmp MessageEntity
		var sendDtm sql.NullTime
		err = r.Scan(&tmp.ReceiverId, &tmp.SenderId, &tmp.Message, &sendDtm)
		if err != nil {
			return nil, err
		}
		if sendDtm.Valid {
			tmp.SendDtm = &sendDtm.Time
		}

		entities = append(entities, tmp)
	}
	return entities, nil
}

func (repo *message) initTable() {
	_, err := repo.db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, receiver_id VARCHAR(50) NOT NULL, sender_id VARCHAR(50) NOT NULL, msg TEXT, is_read CHAR(1), send_dtm datetime, read_dtm datetime)", repo.tableName))
	if err != nil {
		panic(err)
	}
}
