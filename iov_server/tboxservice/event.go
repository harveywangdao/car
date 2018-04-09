package tboxservice

import (
	"errors"
	"hcxy/iov/database"
	"hcxy/iov/log/logger"
	"hcxy/iov/message"
	"time"
)

const (
	TboxUnRegister = iota
	TboxRegisteredUnLogin
	TboxRegisteredLogined
)

const (
	DBName              = "iovdb"
	AesKeyOutOfDateTime = 24 * 60 * 60
)

type Event struct {
	EventMsgChan    chan EventMessage
	Tid             uint32
	Conn            message.MessageConn
	IsNetConnection bool
	bid             uint32

	login      Login
	relogin    ReLogin
	heartbeat  Heartbeat
	readConfig ReadConfig
}

type EventMessage struct {
	Event int
	Msg   *message.Message
	Param interface{}
}

func (eve *Event) SetBid(bid uint32) error {
	eve.bid = bid
	return nil
}

func (eve *Event) GetBid() uint32 {
	return eve.bid
}

func (eve *Event) GetAesKey() (string, error) {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	var tboxaes128key string
	err = tboxDB.QueryRow("SELECT tboxaes128key FROM tboxbaseinfo_tbl WHERE bid = ?", eve.bid).Scan(
		&tboxaes128key)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	return tboxaes128key, nil
}

func (eve *Event) RegisterAckEvent(conn message.MessageConn, recvMsg *message.Message) error {
	err := RegisterACK(conn, recvMsg)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func (eve *Event) getTboxStatusFromDB() (int, error) {
	//Get from DB
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return TboxUnRegister, err
	}

	var status int
	err = tboxDB.QueryRow("SELECT status FROM tboxbaseinfo_tbl WHERE id = ?", 1).Scan(&status)
	if err != nil {
		logger.Error(err)
		return TboxUnRegister, err
	}

	logger.Debug("status =", status)

	return status, nil
}

func (eve *Event) setTboxStatusToDB(status int) error {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	stmtUpd, err := tboxDB.Prepare("UPDATE tboxbaseinfo_tbl SET status=? where id=?")
	if err != nil {
		logger.Error(err)
		return err
	}

	defer stmtUpd.Close()

	_, err = stmtUpd.Exec(status, 1)
	if err != nil {
		logger.Error(err)
		return err
	}

	/*	n, _ := res.RowsAffected()
		logger.Debug("n =", n)*/

	return nil
}

func (eve *Event) eventInit() error {
	return nil
}

func (eve *Event) checkAesKeyOutOfDate(bid uint32) bool {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return false
	}

	var eventcreationtime uint32
	err = tboxDB.QueryRow("SELECT eventcreationtime FROM tboxbaseinfo_tbl WHERE bid = ?", bid).Scan(
		&eventcreationtime)
	if err != nil {
		logger.Error(err)
		return false
	}

	if time.Now().Unix()-int64(eventcreationtime) >= AesKeyOutOfDateTime {
		return true
	}

	return false
}

func (eve *Event) getAes128Key(bid uint32) string {
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

func (eve *Event) taskReadTcp() {
	for {
		msg := message.Message{
			Connection: eve.Conn,
			CallbackFn: eve.getAes128Key,
		}

		errorCode, err := msg.RecvMessage()
		if err != nil {
			logger.Error(err)
			if errorCode == message.ErrorCodeConnectionBreak {
				eve.PushEventChannel(EventConnectionClosed, nil) /*need go out*/
				return
			} else {
				continue
			}
		}

		e := EventMessage{
			Event: GetEventTypeByAidMid(msg.DisPatch.Aid, msg.DisPatch.Mid),
			Msg:   &msg,
		}

		eve.EventMsgChan <- e
	}
}

func (eve *Event) eventDispatcher(eveMsg EventMessage) error {
	logger.Info("event =", GetEventName(eveMsg.Event))

	switch eveMsg.Event {
	case EventConnectionClosed:
		return errors.New("Connection closed!")

	case RegisterReqEventMessage:
		eve.PushEventChannel(RegisterAckEventMessage, eveMsg.Msg)

	case RegisterAckEventMessage:
		err := eve.RegisterAckEvent(eve.Conn, eveMsg.Msg)
		if err != nil {
			logger.Error(err)
			return err
		}

	case EventLoginRequest:
		eve.login.LoginRequest(eve, eveMsg.Msg)

	case EventLoginChallenge:
		eve.login.LoginChallenge(eve, eveMsg.Msg)

	case EventLoginResponse:
		eve.login.LoginResponse(eve, eveMsg.Msg)

	case EventLoginSuccess:
		eve.login.LoginSuccess(eve, eveMsg.Msg)
		//eve.PushEventChannel(EventReLoginRequest, nil)
		eve.PushEventChannel(EventReadConfigRequest, nil)

	case EventLoginFailure:
		eve.login.LoginFailure(eve, eveMsg.Msg)

	case EventReLoginRequest:
		eve.relogin.ReLoginReq(eve)

	case EventReLoginAck:
		eve.relogin.ReLoginAck(eve, eveMsg.Msg)

	case EventHeartbeatRequest:
		eve.heartbeat.HeartbeatReq(eve, eveMsg.Msg)

	case EventHeartbeatAck:
		eve.heartbeat.HeartbeatAck(eve, eveMsg.Msg)

	case EventReadConfigRequest:
		eve.readConfig.ReadConfigReq(eve)

	case EventReadConfigAck:
		eve.readConfig.ReadConfigAck(eve, eveMsg.Msg)

	default:
		logger.Error("Unknown event!")
	}

	return nil
}

func (eve *Event) EventScheduler() error {
	defer eve.eventDestory()

	err := eve.eventInit()
	if err != nil {
		logger.Error(err)
		return err
	}

	if eve.IsNetConnection {
		go eve.taskReadTcp()
	}

	for {
		select {
		case eveMsg := <-eve.EventMsgChan:
			logger.Debug("eveMsg =", eveMsg)
			err = eve.eventDispatcher(eveMsg)
			if err != nil {
				logger.Error(err)
				return err
			}

		case <-time.After(time.Second * 60):
			logger.Debug("Timeout!")
		}
	}

	return nil
}

func (eve *Event) PushEventChannel(event int, msg *message.Message) {
	e := EventMessage{
		Event: event,
		Msg:   msg,
	}

	eve.EventMsgChan <- e
}

func (eve *Event) PushEventChannel2(event int, msg *message.Message, param interface{}) {
	e := EventMessage{
		Event: event,
		Msg:   msg,
		Param: param,
	}

	eve.EventMsgChan <- e
}

func (eve *Event) eventDestory() error {
	return nil
}

func NewEvent(eveMsgChan chan EventMessage, tid uint32, conn message.MessageConn, isNetConnection bool) (*Event, error) {
	eve := Event{}
	eve.Conn = conn
	eve.EventMsgChan = eveMsgChan
	eve.Tid = tid
	eve.IsNetConnection = isNetConnection

	return &eve, nil
}
