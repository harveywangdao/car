package tbox

import (
	"errors"
	"hcxy/iov/database"
	"hcxy/iov/log/logger"
	"hcxy/iov/message"
	"net"
	"time"
)

const (
	TboxUnRegister = iota
	TboxRegisteredUnLogin
	TboxRegisteredLogined
)

const (
	DBName                        = "tboxdb"
	RegisterReqMaxTimes           = 3
	AesValidityDurationTime       = 24*60*60 - 10*60 /*24hours - 10minute*/
	AesKeyOutOfDateTime           = 24 * 60 * 60
	CheckAesKeyValidityTickerTime = 5 * time.Minute
)

type Event struct {
	EventMsgChan chan EventMessage
	IPPort       string
	TboxStatus   int
	TboxNo       int

	Conn net.Conn

	checkAesKeyValidityTicker *time.Ticker

	account           Account
	login             Login
	relogin           ReLogin
	heartbeat         Heartbeat
	readConfig        ReadConfig
	vehicleControl    VehicleControl
	vehicleInfoUpload VehicleInfoUpload
}

type EventMessage struct {
	Event int
	Msg   *message.Message
}

func (eve *Event) GetAesKey() (string, error) {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	var tboxaes128key string
	err = tboxDB.QueryRow("SELECT tboxaes128key FROM tboxfactorydata_tbl WHERE id = ?", eve.TboxNo).Scan(
		&tboxaes128key)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	return tboxaes128key, nil
}

func (eve *Event) GetBid() uint32 {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return 0
	}

	var bid uint32
	err = tboxDB.QueryRow("SELECT bid FROM tboxfactorydata_tbl WHERE id = ?", eve.TboxNo).Scan(
		&bid)
	if err != nil {
		logger.Error(err)
		return 0
	}

	return bid
}

func (eve *Event) getTboxStatusFromDB() (int, error) {
	//Get from DB
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return TboxUnRegister, err
	}

	var status int
	err = tboxDB.QueryRow("SELECT status FROM tboxfactorydata_tbl WHERE id = ?", 1).Scan(&status)
	if err != nil {
		logger.Error(err)
		return TboxUnRegister, err
	}

	logger.Debug("status =", status)

	return status, nil
}

func (eve *Event) SetTboxStatusToDB(status int) error {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	stmtUpd, err := tboxDB.Prepare("UPDATE tboxfactorydata_tbl SET status=? where id=?")
	if err != nil {
		logger.Error(err)
		return err
	}

	defer stmtUpd.Close()

	_, err = stmtUpd.Exec(status, eve.TboxNo)
	if err != nil {
		logger.Error(err)
		return err
	}

	/*	n, _ := res.RowsAffected()
		logger.Debug("n =", n)*/

	return nil
}

func (eve *Event) getAes128Key(bid uint32) string {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return ""
	}

	var pretboxaes128key, tboxaes128key string
	var eventcreationtime uint32
	err = tboxDB.QueryRow("SELECT pretboxaes128key,tboxaes128key,eventcreationtime FROM tboxfactorydata_tbl WHERE id = ?", eve.TboxNo).Scan(
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

func (eve *Event) checkAesKeyValidity() error {
	eventCreationTime, err := GetEventCreationTime(1)
	if err != nil {
		logger.Error(err)
		return err
	}

	if time.Now().Unix()-int64(eventCreationTime) > AesValidityDurationTime {
		logger.Debug("Aes key timeout, need relogin.")
		eve.TboxStatus = TboxRegisteredUnLogin
		eve.SetTboxStatusToDB(TboxRegisteredUnLogin)
		eve.PushEventChannel(EventLoginRequest, nil)
	}

	return nil
}

func (eve *Event) eventInit() error {
	status, err := eve.getTboxStatusFromDB()
	if err != nil {
		logger.Error(err)
		return err
	}

	eve.TboxStatus = status

	eve.checkAesKeyValidityTicker = time.NewTicker(CheckAesKeyValidityTickerTime)

	return nil
}

func (eve *Event) connectServer() error {
	conn, err := net.Dial("tcp", eve.IPPort)
	if err != nil {
		logger.Error("Dial fail!", err)
		eve.PushEventChannel(EventConnectServer, nil)
		return err
	}

	eve.Conn = conn

	go eve.taskReadTcp()

	eve.PushEventChannel(EventCheckTboxStatus, nil)

	return nil
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
				eve.PushEventChannel(EventConnectServer, nil) /*need connect again.*/
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
	case EventCheckTboxStatus:
		if true { //eve.TboxStatus == TboxUnRegister {
			eve.PushEventChannel(RegisterReqEventMessage, nil)
		} else {
			if eve.TboxStatus == TboxRegisteredUnLogin {
				eve.PushEventChannel(EventLoginRequest, nil)
			} else {
				eve.checkAesKeyValidity()
			}
		}

	case EventConnectServer:
		eve.connectServer()

	case EventServerClosed:
		return errors.New("Server closed!")

		/*	case EventSaveTboxStatusUnRegister:
				eve.SetTboxStatusToDB(TboxUnRegister)

			case EventSaveTboxStatusUnLogin:
				eve.SetTboxStatusToDB(TboxRegisteredUnLogin)
		*/
	case EventSaveTboxStatusLogined:
		eve.SetTboxStatusToDB(TboxRegisteredLogined)

	case RegisterReqEventMessage:
		eve.TboxStatus = TboxUnRegister
		eve.SetTboxStatusToDB(TboxUnRegister)
		eve.account.RegisterReq(eve)

	case RegisterAckEventMessage:
		eve.account.RegisterACK(eve, eveMsg.Msg)

	case EventLoginRequest:
		eve.login.LoginRequest(eve)

	case EventLoginChallenge:
		eve.login.LoginChallenge(eve, eveMsg.Msg)

	case EventLoginResponse:
		eve.login.LoginResponse(eve, eveMsg.Msg)

	case EventLoginSuccess:
		eve.login.LoginSuccess(eve, eveMsg.Msg)
		eve.PushEventChannel(EventHeartbeatRequest, nil)

	case EventLoginFailure:
		eve.login.LoginFailure(eve, eveMsg.Msg)

	case EventReLoginRequest:
		eve.relogin.ReLoginReq(eve, eveMsg.Msg)

	case EventReLoginAck:
		eve.relogin.ReLoginAck(eve, eveMsg.Msg)

	case EventHeartbeatRequest:
		eve.heartbeat.HeartbeatReq(eve)

	case EventHeartbeatAck:
		eve.heartbeat.HeartbeatAck(eve, eveMsg.Msg)

	case EventReadConfigRequest:
		eve.readConfig.ReadConfigReq(eve, eveMsg.Msg)

	case EventReadConfigAck:
		eve.readConfig.ReadConfigAck(eve, eveMsg.Msg)

	case EventRemoteOperationRequest:
		eve.vehicleControl.RemoteOperationReq(eve, eveMsg.Msg)

	case EventDispatcherAckMessage:
		eve.vehicleControl.DispatcherAckMessage2(eve, eveMsg.Msg)

	case EventRemoteOperationEnd:
		eve.vehicleControl.RemoteOperationEnd(eve, eveMsg.Msg)

	case EventRemoteOperationAck:
		eve.vehicleControl.RemoteOperationAck(eve, eveMsg.Msg)

	case EventVHSUpdateMessage:
		eve.vehicleInfoUpload.VHSUpdateMessage(eve)

	case EventVHSUpdateMessageAck:
		eve.vehicleInfoUpload.VHSUpdateMessageAck(eve, eveMsg.Msg)

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

	eve.PushEventChannel(EventConnectServer, nil)

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

		case <-eve.checkAesKeyValidityTicker.C:
			logger.Debug("----checkAesKeyValidityTicker----")
			eve.checkAesKeyValidity()
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

func (eve *Event) eventDestory() error {
	return nil
}

func NewEvent(eveMsgChan chan EventMessage, ipport string, tboxNo int) (*Event, error) {
	eve := Event{}

	eve.IPPort = ipport
	eve.EventMsgChan = eveMsgChan
	eve.TboxNo = tboxNo

	return &eve, nil
}
