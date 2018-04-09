package tboxservice

import (
	"hcxy/iov/database"
	"hcxy/iov/log/logger"
	"hcxy/iov/message"
	"hcxy/iov/util"
	"time"
)

const (
	TboxIdLen = 4
)

type TboxDispatcher struct {
	MqToTboxChan chan []byte
	TboxToMqChan chan []byte

	TidMsg     chan *TboxIdMsg
	tboxTidMap map[uint32]chan EventMessage
}

type TboxIdMsg struct {
	Tid uint32
	Msg *message.Message
}

type TboxMqConnection struct {
	TboxToMqChan chan []byte
	Tid          uint32
}

func (tmp TboxMqConnection) Read(b []byte) (n int, err error) {
	logger.Error("Not Support!")
	return 0, nil
}

func (tmp TboxMqConnection) Write(b []byte) (n int, err error) {
	data := addTidHeader(tmp.Tid, b)

	tmp.TboxToMqChan <- data
	return len(b), nil
}

func addTidHeader(tid uint32, data []byte) []byte {
	tidHeaderData, err := util.Uint32ToByteSlice(tid)
	if err != nil {
		logger.Error(err)
		return data
	}

	tidHeaderData = append(tidHeaderData, data...)

	return tidHeaderData
}

func getTidHeader(data []byte) uint32 {
	tid, err := util.ByteSliceToUint32(data[:TboxIdLen])
	if err != nil {
		logger.Error(err)
		return 0
	}

	return tid
}

func (r *TboxDispatcher) getAes128Key(bid uint32) string {
	logger.Debug("bid = ", bid)
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return ""
	}

	var pretboxaes128key, tboxaes128key string
	var eventcreationtime uint32
	err = tboxDB.QueryRow("SELECT pretboxaes128key,tboxaes128key,eventcreationtime FROM tboxbaseinfo_tbl WHERE bid = ?", bid).Scan(
		&pretboxaes128key,
		&tboxaes128key,
		&eventcreationtime)
	if err != nil {
		logger.Error(err)
		return ""
	}

	if time.Now().Unix()-int64(eventcreationtime) >= AesKeyOutOfDateTime {
		return pretboxaes128key
	}

	return tboxaes128key
}

func (r *TboxDispatcher) recvFromMq(tidMsg chan *TboxIdMsg) {
	for {
		data, ok := <-r.MqToTboxChan
		if !ok {
			logger.Error("MqToTboxChan Close!")
			return
		}

		msg := &message.Message{
			/*Connection: eve.Conn,*/
			CallbackFn: r.getAes128Key,
		}

		logger.Debug("data =", data)

		_, err := msg.ParseOneMessage(data[TboxIdLen:])
		if err != nil {
			logger.Error(err)
			continue
		}

		tm := &TboxIdMsg{
			Tid: getTidHeader(data),
			Msg: msg,
		}

		tidMsg <- tm
	}
}

func (r *TboxDispatcher) tboxServiceHandler(eventMsgChan chan EventMessage, tid uint32, conn message.MessageConn) {
	eve, err := NewEvent(eventMsgChan, tid, conn, false)
	if err != nil {
		logger.Error(err)
		return
	}

	err = eve.EventScheduler()
	if err != nil {
		logger.Error(err)
		return
	}
}

func (r *TboxDispatcher) pushEventMessage(eventMsgChan chan EventMessage, msg *message.Message) {
	e := EventMessage{
		Event: GetEventTypeByAidMid(msg.DisPatch.Aid, msg.DisPatch.Mid),
		Msg:   msg,
	}

	eventMsgChan <- e
}

func (r *TboxDispatcher) demux(tidMsg *TboxIdMsg) error {
	logger.Debug("tidMsg =", *tidMsg)

	ch, ok := r.tboxTidMap[tidMsg.Tid]
	if ok {
		r.pushEventMessage(ch, tidMsg.Msg)
	} else {
		eventMsgChan := make(chan EventMessage, 128)

		r.tboxTidMap[tidMsg.Tid] = eventMsgChan

		r.pushEventMessage(eventMsgChan, tidMsg.Msg)

		tboxMqConn := TboxMqConnection{
			TboxToMqChan: r.TboxToMqChan,
			Tid:          tidMsg.Tid,
		}

		go r.tboxServiceHandler(eventMsgChan, tidMsg.Tid, tboxMqConn)
	}

	return nil
}

func (r *TboxDispatcher) Start() error {
	r.tboxTidMap = make(map[uint32]chan EventMessage)
	r.TidMsg = make(chan *TboxIdMsg, 128)

	go r.recvFromMq(r.TidMsg)

	for {
		select {
		case tidMsg := <-r.TidMsg:
			err := r.demux(tidMsg)
			if err != nil {
				logger.Error(err)
			}
		}
	}

	return nil
}
