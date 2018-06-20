package thing

import (
	"errors"
	"github.com/harveywangdao/road/database"
	"github.com/harveywangdao/road/log/logger"
	"github.com/harveywangdao/road/message"
	"net"
	"time"
)

const (
	ThingUnRegister = iota
	ThingRegisteredUnLogin
	ThingRegisteredLogined
)

const (
	DBName                        = "thingsdb"
	RegisterReqMaxTimes           = 3
	AesValidityDurationTime       = 24*60*60 - 10*60 /*24hours - 10minute*/
	AesKeyOutOfDateTime           = 24 * 60 * 60
	CheckAesKeyValidityTickerTime = 5 * time.Minute
	ConnectServerDelayTime        = 5 * time.Second
)

type Thing struct {
	ThingMsgChan chan ThingMessage
	IPPort       string
	ThingStatus  int
	ThingNo      int

	Conn net.Conn

	checkAesKeyValidityTicker *time.Ticker

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
}

func (thing *Thing) GetAesKey() (string, error) {
	thingDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	var thingaes128key string
	err = thingDB.QueryRow("SELECT thingaes128key FROM thingbaseinfodata_tbl WHERE id = ?", thing.ThingNo).Scan(
		&thingaes128key)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	return thingaes128key, nil
}

func (thing *Thing) GetBid() uint32 {
	thingDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return 0
	}

	var bid uint32
	err = thingDB.QueryRow("SELECT bid FROM thingbaseinfodata_tbl WHERE id = ?", thing.ThingNo).Scan(
		&bid)
	if err != nil {
		logger.Error(err)
		return 0
	}

	return bid
}

func (thing *Thing) getThingStatusFromDB() (int, error) {
	//Get from DB
	db, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return ThingUnRegister, err
	}

	var status int
	err = db.QueryRow("SELECT status FROM thingbaseinfodata_tbl WHERE id = ?", thing.ThingNo).Scan(&status)
	if err != nil {
		logger.Error(err)
		return ThingUnRegister, err
	}

	logger.Debug("status =", status)

	return status, nil
}

func (thing *Thing) SetThingStatusToDB(status int) error {
	db, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("status =", status)
	stmtUpd, err := db.Prepare("UPDATE thingbaseinfodata_tbl SET status=? where id=?")
	if err != nil {
		logger.Error(err)
		return err
	}

	defer stmtUpd.Close()

	_, err = stmtUpd.Exec(status, thing.ThingNo)
	if err != nil {
		logger.Error(err)
		return err
	}

	/*	n, _ := res.RowsAffected()
		logger.Debug("n =", n)*/

	return nil
}

func (thing *Thing) getAes128Key(bid uint32) string {
	db, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return ""
	}

	var prethingaes128key, thingaes128key string
	var eventcreationtime uint32
	err = db.QueryRow("SELECT prethingaes128key,thingaes128key,eventcreationtime FROM thingbaseinfodata_tbl WHERE id = ?", thing.ThingNo).Scan(
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

func (thing *Thing) checkAesKeyValidity() error {
	eventCreationTime, err := GetEventCreationTime(1)
	if err != nil {
		logger.Error(err)
		return err
	}

	if time.Now().Unix()-int64(eventCreationTime) > AesValidityDurationTime {
		logger.Debug("Aes key timeout, need relogin.")
		thing.ThingStatus = ThingRegisteredUnLogin
		thing.SetThingStatusToDB(ThingRegisteredUnLogin)
		thing.PushEventChannel(EventLoginRequest, nil)
	}

	return nil
}

func (thing *Thing) tboxInit() error {
	status, err := thing.getThingStatusFromDB()
	if err != nil {
		logger.Error(err)
		return err
	}

	thing.ThingStatus = status

	thing.checkAesKeyValidityTicker = time.NewTicker(CheckAesKeyValidityTickerTime)

	return nil
}

func (thing *Thing) connectServer() error {
	conn, err := net.Dial("tcp", thing.IPPort)
	if err != nil {
		logger.Error("Dial fail!", err)

		connectServerDelayTimer := time.NewTimer(ConnectServerDelayTime)

		go func() {
			logger.Debug("Start timer......")
			select {
			case <-connectServerDelayTimer.C:
				logger.Info("Connect server again!")
				thing.PushEventChannel(EventConnectServer, nil)
			}
		}()

		return err
	}

	logger.Debug("Net =", conn.RemoteAddr().Network(), ", Addr =", conn.RemoteAddr().String())

	thing.Conn = conn

	go thing.taskReadTcp()

	thing.PushEventChannel(EventCheckThingStatus, nil)

	return nil
}

func (thing *Thing) setStatusWhenBroken() error {
	if thing.ThingStatus == ThingRegisteredLogined {
		thing.ThingStatus = ThingRegisteredUnLogin
		thing.SetThingStatusToDB(ThingRegisteredUnLogin)
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
				thing.setStatusWhenBroken()
				thing.PushEventChannel(EventConnectServer, nil) /*need connect again.*/
				return
			} else {
				continue
			}
		}

		e := ThingMessage{
			Event: GetEventTypeByAidMid(msg.DisPatch.Aid, msg.DisPatch.Mid),
			Msg:   &msg,
		}

		thing.ThingMsgChan <- e
	}
}

func (thing *Thing) eventDispatcher(thingMsg ThingMessage) error {
	logger.Debug("event =", GetEventName(thingMsg.Event))

	switch thingMsg.Event {
	case EventCheckThingStatus:
		if true { //thing.ThingStatus == ThingUnRegister {
			thing.PushEventChannel(RegisterReqEventMessage, nil)
		} else {
			if thing.ThingStatus == ThingRegisteredUnLogin {
				thing.PushEventChannel(EventLoginRequest, nil)
			} else {
				thing.checkAesKeyValidity()
			}
		}

	case EventConnectServer:
		thing.connectServer()

	case EventServerClosed:
		return errors.New("Server closed!")

	case RegisterReqEventMessage:
		thing.ThingStatus = ThingUnRegister
		thing.SetThingStatusToDB(ThingUnRegister)
		thing.register.RegisterReq(thing)

	case RegisterAckEventMessage:
		thing.register.RegisterACK(thing, thingMsg.Msg)

	case EventLoginRequest:
		thing.login.LoginRequest(thing)

	case EventLoginChallenge:
		thing.login.LoginChallenge(thing, thingMsg.Msg)

	case EventLoginResponse:
		thing.login.LoginResponse(thing, thingMsg.Msg)

	case EventLoginSuccess:
		thing.login.LoginSuccess(thing, thingMsg.Msg)
		//thing.PushEventChannel(EventHeartbeatRequest, nil)
		//thing.PushEventChannel(EventThingInfoUpload, nil)

	case EventLoginFailure:
		thing.login.LoginFailure(thing, thingMsg.Msg)

	case EventReLoginRequest:
		thing.relogin.ReLoginReq(thing, thingMsg.Msg)

	case EventReLoginAck:
		thing.relogin.ReLoginAck(thing, thingMsg.Msg)

	case EventHeartbeatRequest:
		thing.heartbeat.HeartbeatReq(thing)

	case EventHeartbeatAck:
		thing.heartbeat.HeartbeatAck(thing, thingMsg.Msg)

	case EventReadConfigRequest:
		thing.readConfig.ReadConfigReq(thing, thingMsg.Msg)

	case EventReadConfigAck:
		thing.readConfig.ReadConfigAck(thing, thingMsg.Msg)

	case EventSetConfigRequest:
		thing.setConfig.SetConfigReq(thing, thingMsg.Msg)

	case EventSetConfigAck:
		thing.setConfig.SetConfigAck(thing, thingMsg.Msg)

	case EventRemoteOperationRequest:
		thing.thingControl.RemoteOperationReq(thing, thingMsg.Msg)

	case EventDispatcherAckMessage:
		thing.thingControl.DispatcherAckMessage2(thing, thingMsg.Msg)

	case EventRemoteOperationEnd:
		thing.thingControl.RemoteOperationEnd(thing, thingMsg.Msg)

	case EventRemoteOperationAck:
		thing.thingControl.RemoteOperationAck(thing, thingMsg.Msg)

	case EventThingInfoUpload:
		thing.thingInfoUpload.ThingInfoUploadReq(thing)

	case EventThingInfoUploadAck:
		thing.thingInfoUpload.ThingInfoUploadAck(thing, thingMsg.Msg)

	default:
		logger.Error("Unknown event!")
	}

	return nil
}

func (thing *Thing) ThingScheduler() error {
	defer thing.eventDestory()

	err := thing.tboxInit()
	if err != nil {
		logger.Error(err)
		return err
	}

	thing.PushEventChannel(EventConnectServer, nil)

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

		case <-thing.checkAesKeyValidityTicker.C:
			logger.Debug("----checkAesKeyValidityTicker----")
			thing.checkAesKeyValidity()
		}
	}

	return nil
}

func (thing *Thing) PushEventChannel(event int, msg *message.Message) {
	e := ThingMessage{
		Event: event,
		Msg:   msg,
	}

	thing.ThingMsgChan <- e
}

func (thing *Thing) eventDestory() error {
	return nil
}

func NewThing(thingMsgChan chan ThingMessage, ipport string, thingNo int) (*Thing, error) {
	thing := Thing{}

	thing.IPPort = ipport
	thing.ThingMsgChan = thingMsgChan
	thing.ThingNo = thingNo

	return &thing, nil
}
