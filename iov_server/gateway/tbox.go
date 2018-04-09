package gateway

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

type Tbox struct {
	TboxMsgChan  chan TboxMessage
	Conn         message.MessageConn
	TboxConnChan chan TboxConn

	bid uint32
	vin string

	register          Register
	login             Login
	relogin           ReLogin
	heartbeat         Heartbeat
	readConfig        ReadConfig
	vehicleControl    VehicleControl
	vehicleInfoUpload VehicleInfoUpload
}

type TboxMessage struct {
	Event int
	Msg   *message.Message
	Param interface{}
}

func (tbox *Tbox) SetVinAndBid(vin string, bid uint32) error {
	tbox.vin = vin
	tbox.bid = bid

	tboxConn := TboxConn{
		VIN:         tbox.vin,
		TboxService: tbox,
	}
	tbox.TboxConnChan <- tboxConn

	return nil
}

func (tbox *Tbox) GetBid() uint32 {
	return tbox.bid
}

func (tbox *Tbox) GetAesKey() (string, error) {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	var tboxaes128key string
	err = tboxDB.QueryRow("SELECT tboxaes128key FROM tboxbaseinfo_tbl WHERE bid = ?", tbox.bid).Scan(
		&tboxaes128key)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	return tboxaes128key, nil
}

func (tbox *Tbox) tboxInit() error {
	return nil
}

func (tbox *Tbox) CheckAesKeyOutOfDate(bid uint32) bool {
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

func (tbox *Tbox) getAes128Key(bid uint32) string {
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

func (tbox *Tbox) taskReadTcp() {
	for {
		msg := message.Message{
			Connection: tbox.Conn,
			CallbackFn: tbox.getAes128Key,
		}

		errorCode, err := msg.RecvMessage()
		if err != nil {
			logger.Error(err)
			if errorCode == message.ErrorCodeConnectionBreak {
				tbox.PushEventChannel(EventConnectionClosed, nil) /*need go out*/
				return
			} else {
				continue
			}
		}

		tm := TboxMessage{
			Event: GetEventTypeByAidMid(msg.DisPatch.Aid, msg.DisPatch.Mid),
			Msg:   &msg,
		}

		tbox.TboxMsgChan <- tm
	}
}

func (tbox *Tbox) eventDispatcher(tboxMsg TboxMessage) error {
	logger.Info("event =", GetEventName(tboxMsg.Event))

	switch tboxMsg.Event {
	case EventConnectionClosed:
		return errors.New("Connection closed!")

	case RegisterReqEventMessage:
		tbox.PushEventChannel(RegisterAckEventMessage, tboxMsg.Msg)

	case RegisterAckEventMessage:
		tbox.register.RegisterACK(tbox.Conn, tboxMsg.Msg)

	case EventLoginRequest:
		tbox.login.LoginRequest(tbox, tboxMsg.Msg)

	case EventLoginChallenge:
		tbox.login.LoginChallenge(tbox, tboxMsg.Msg)

	case EventLoginResponse:
		tbox.login.LoginResponse(tbox, tboxMsg.Msg)

	case EventLoginSuccess:
		tbox.login.LoginSuccess(tbox, tboxMsg.Msg)
		//tbox.PushEventChannel(EventReLoginRequest, nil)
		tbox.PushEventChannel(EventReadConfigRequest, nil)

	case EventLoginFailure:
		tbox.login.LoginFailure(tbox, tboxMsg.Msg)

	case EventReLoginRequest:
		tbox.relogin.ReLoginReq(tbox)

	case EventReLoginAck:
		tbox.relogin.ReLoginAck(tbox, tboxMsg.Msg)

	case EventHeartbeatRequest:
		tbox.heartbeat.HeartbeatReq(tbox, tboxMsg.Msg)

	case EventHeartbeatAck:
		tbox.heartbeat.HeartbeatAck(tbox, tboxMsg.Msg)

	case EventReadConfigRequest:
		tbox.readConfig.ReadConfigReq(tbox)

	case EventReadConfigAck:
		tbox.readConfig.ReadConfigAck(tbox, tboxMsg.Msg)

	case EventRemoteOperationRequest:
		tbox.vehicleControl.RemoteOperationReq(tbox, tboxMsg.Param.(string))

	case EventDispatcherAckMessage:
		tbox.vehicleControl.DispatcherAckMessage(tbox, tboxMsg.Msg)

	case EventRemoteOperationEnd:
		tbox.vehicleControl.RemoteOperationEnd(tbox, tboxMsg.Msg)

	case EventRemoteOperationAck:
		tbox.vehicleControl.RemoteOperationAck(tbox, tboxMsg.Msg)

	case EventVHSUpdateMessage:
		tbox.vehicleInfoUpload.VHSUpdateMessage(tbox, tboxMsg.Msg)

	case EventVHSUpdateMessageAck:
		tbox.vehicleInfoUpload.VHSUpdateMessageAck(tbox, tboxMsg.Msg)

	default:
		logger.Error("Unknown event!")
	}

	return nil
}

func (tbox *Tbox) TboxScheduler() error {
	defer tbox.tboxDestory()

	err := tbox.tboxInit()
	if err != nil {
		logger.Error(err)
		return err
	}

	go tbox.taskReadTcp()

	for {
		select {
		case tboxMsg := <-tbox.TboxMsgChan:
			logger.Debug("tboxMsg =", tboxMsg)
			err = tbox.eventDispatcher(tboxMsg)
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

func (tbox *Tbox) PushEventChannel(event int, msg *message.Message) {
	tm := TboxMessage{
		Event: event,
		Msg:   msg,
	}

	tbox.TboxMsgChan <- tm
}

func (tbox *Tbox) PushEventChannel2(event int, msg *message.Message, param interface{}) {
	tm := TboxMessage{
		Event: event,
		Msg:   msg,
		Param: param,
	}

	tbox.TboxMsgChan <- tm
}

func (tbox *Tbox) tboxDestory() error {
	return nil
}

func NewTbox(msgChan chan TboxMessage, conn message.MessageConn, tboxConnChan chan TboxConn) (*Tbox, error) {
	tbox := Tbox{}
	tbox.Conn = conn
	tbox.TboxMsgChan = msgChan
	tbox.TboxConnChan = tboxConnChan

	return &tbox, nil
}
