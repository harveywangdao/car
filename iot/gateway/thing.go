package gateway

import (
	"errors"
	"github.com/harveywangdao/road/database"
	"github.com/harveywangdao/road/log/logger"
	"github.com/harveywangdao/road/message"
	"time"
)

const (
	ThingUnRegister = iota
	ThingRegisteredUnLogin
	ThingRegisteredLogined
)

const (
	DBName              = "iotdb"
	AesKeyOutOfDateTime = 24 * 60 * 60
)

type Thing struct {
	ThingMsgChan        chan ThingMessage
	Conn                message.MessageConn
	AddThingConnChan    chan ThingConn
	DeleteThingConnChan chan ThingConn

	bid     uint32
	thingid string

	register        Register
	login           Login
	relogin         ReLogin
	heartbeat       Heartbeat
	readConfig      ReadConfig
	setConfig       SetConfig
	thingControl    ThingControl
	thingInfoUpload ThingInfoUpload
}

type ThingMessage struct {
	Event int
	Msg   *message.Message
	Param interface{}
}

func (thing *Thing) SetThingIdAndBid(thingid string, bid uint32) error {
	thing.thingid = thingid
	thing.bid = bid

	ThingConn := ThingConn{
		ThingID:      thing.thingid,
		ThingService: thing,
	}
	thing.AddThingConnChan <- ThingConn

	return nil
}

func (thing *Thing) GetBid() uint32 {
	return thing.bid
}

func (thing *Thing) GetAesKey() (string, error) {
	db, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	var thingaes128key string
	err = db.QueryRow("SELECT thingaes128key FROM thingbaseinfodata_tbl WHERE thingid = ?", thing.thingid).Scan(
		&thingaes128key)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	return thingaes128key, nil
}

func (thing *Thing) thingInit() error {
	return nil
}

func (thing *Thing) CheckAesKeyOutOfDate(bid uint32) bool {
	db, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return false
	}

	var eventcreationtime uint32
	err = db.QueryRow("SELECT eventcreationtime FROM thingbaseinfodata_tbl WHERE thingid = ?", thing.thingid).Scan(
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

func (thing *Thing) getAes128Key(bid uint32) string {
	db, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return ""
	}

	var prethingaes128key, thingaes128key string
	var eventcreationtime uint32
	err = db.QueryRow("SELECT prethingaes128key,thingaes128key,eventcreationtime FROM thingbaseinfodata_tbl WHERE bid = ?", bid).Scan(
		&prethingaes128key,
		&thingaes128key,
		&eventcreationtime)
	if err != nil {
		logger.Error(err)
		return ""
	}

	if time.Now().Unix()-int64(eventcreationtime) >= AesKeyOutOfDateTime {
		return prethingaes128key
	}

	return thingaes128key
}

func (thing *Thing) saveTboxState(status int) error {
	db, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	stmtUpd, err := db.Prepare("UPDATE thingbaseinfodata_tbl SET status=? where thingid=?")
	if err != nil {
		logger.Error(err)
		return err
	}
	defer stmtUpd.Close()

	_, err = stmtUpd.Exec(status, thing.thingid)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func (thing *Thing) taskReadTcp() {
	for {
		msg := message.Message{
			Connection: thing.Conn,
			CallbackFn: thing.getAes128Key,
		}

		errorCode, err := msg.RecvMessage()
		if err != nil {
			logger.Error(err)
			if errorCode == message.ErrorCodeConnectionBreak {
				thing.PushEventChannel(EventConnectionClosed, nil) /*need go out*/
				return
			} else {
				continue
			}
		}

		tm := ThingMessage{
			Event: GetEventTypeByAidMid(msg.DisPatch.Aid, msg.DisPatch.Mid),
			Msg:   &msg,
		}

		thing.ThingMsgChan <- tm
	}
}

func (thing *Thing) destoryThing() error {
	thing.saveTboxState(ThingRegisteredUnLogin)

	ThingConn := ThingConn{
		ThingID:      thing.thingid,
		ThingService: thing,
	}
	thing.DeleteThingConnChan <- ThingConn

	return nil
}

func (thing *Thing) eventDispatcher(thingMsg ThingMessage) error {
	logger.Debug("event =", GetEventName(thingMsg.Event))

	switch thingMsg.Event {
	case EventConnectionClosed:
		thing.destoryThing()
		return errors.New("Connection closed!")

	case RegisterReqEventMessage:
		thing.PushEventChannel(RegisterAckEventMessage, thingMsg.Msg)

	case RegisterAckEventMessage:
		thing.register.RegisterACK(thing.Conn, thingMsg.Msg)

	case EventLoginRequest:
		thing.login.LoginRequest(thing, thingMsg.Msg)

	case EventLoginChallenge:
		thing.login.LoginChallenge(thing, thingMsg.Msg)

	case EventLoginResponse:
		thing.login.LoginResponse(thing, thingMsg.Msg)

	case EventLoginSuccess:
		thing.login.LoginSuccess(thing, thingMsg.Msg)
		//thing.PushEventChannel(EventReLoginRequest, nil)
		//thing.PushEventChannel(EventReadConfigRequest, nil)
		//thing.PushEventChannel(EventSetConfigRequest, nil)
		//thing.PushEventChannel2(EventRemoteOperationRequest, nil, "lock")

	case EventLoginFailure:
		thing.login.LoginFailure(thing, thingMsg.Msg)

	case EventReLoginRequest:
		thing.relogin.ReLoginReq(thing)

	case EventReLoginAck:
		thing.relogin.ReLoginAck(thing, thingMsg.Msg)

	case EventHeartbeatRequest:
		thing.heartbeat.HeartbeatReq(thing, thingMsg.Msg)

	case EventHeartbeatAck:
		thing.heartbeat.HeartbeatAck(thing, thingMsg.Msg)

	case EventReadConfigRequest:
		thing.readConfig.ReadConfigReq(thing)

	case EventReadConfigAck:
		thing.readConfig.ReadConfigAck(thing, thingMsg.Msg)

	case EventSetConfigRequest:
		thing.setConfig.SetConfigReq(thing)

	case EventSetConfigAck:
		thing.setConfig.SetConfigAck(thing, thingMsg.Msg)

	case EventRemoteOperationRequest:
		thing.thingControl.RemoteOperationReq(thing, thingMsg.Param.(string))

	case EventDispatcherAckMessage:
		thing.thingControl.DispatcherAckMessage(thing, thingMsg.Msg)

	case EventRemoteOperationEnd:
		thing.thingControl.RemoteOperationEnd(thing, thingMsg.Msg)

	case EventRemoteOperationAck:
		thing.thingControl.RemoteOperationAck(thing, thingMsg.Msg)

	case EventThingInfoUpload:
		thing.thingInfoUpload.ThingInfoUploadReq(thing, thingMsg.Msg)

	case EventThingInfoUploadAck:
		thing.thingInfoUpload.ThingInfoUploadAck(thing, thingMsg.Msg)

	default:
		logger.Error("Unknown event!")
	}

	return nil
}

func (thing *Thing) ThingScheduler() error {
	defer thing.thingDestory()

	err := thing.thingInit()
	if err != nil {
		logger.Error(err)
		return err
	}

	go thing.taskReadTcp()

	for {
		select {
		case thingMsg := <-thing.ThingMsgChan:
			logger.Debug("thingMsg =", thingMsg)
			err = thing.eventDispatcher(thingMsg)
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

func (thing *Thing) PushEventChannel(event int, msg *message.Message) {
	tm := ThingMessage{
		Event: event,
		Msg:   msg,
	}

	thing.ThingMsgChan <- tm
}

func (thing *Thing) PushEventChannel2(event int, msg *message.Message, param interface{}) {
	tm := ThingMessage{
		Event: event,
		Msg:   msg,
		Param: param,
	}

	thing.ThingMsgChan <- tm
}

func (thing *Thing) thingDestory() error {
	return nil
}

func NewThing(msgChan chan ThingMessage, conn message.MessageConn, addThingConnChan, delThingConnChan chan ThingConn) (*Thing, error) {
	thing := Thing{}
	thing.Conn = conn
	thing.ThingMsgChan = msgChan
	thing.AddThingConnChan = addThingConnChan
	thing.DeleteThingConnChan = delThingConnChan

	return &thing, nil
}
