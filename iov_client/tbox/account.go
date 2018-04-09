package tbox

import (
	"encoding/json"
	"errors"
	"hcxy/iov/database"
	"hcxy/iov/log/logger"
	"hcxy/iov/message"
	"hcxy/iov/util"
	"time"
)

const (
	RegisterSuccess byte = 0x78
	RegisterFailure byte = 0xA8
	AlreadyRegister byte = 0x79

	CheckRegAckTimerTime   = 1 * time.Second
	CheckRegAgainTimerTime = 5 * time.Second
)

const (
	RegisterStatusStop = iota
	RegisterStatusStart
)

type Account struct {
	checkRegAckTimer *time.Timer
	regReqTimes      int
	closeRegAckTimer chan bool

	registerStatus int

	registerStart bool
}

type RegisterReqData struct {
	PerAesKey  uint16
	VIN        string
	TBoxSN     string
	IMSI       string
	RollNumber uint16
	ICCID      string
}

type RegisterAckMsg struct {
	Status      byte   `json:"status"`
	CallbackNum string `json:"callbacknumber"`
	Bid         uint32 `json:"bid"`
}

func (ac *Account) saveRegisterDataToDB(bid, eventCreationTime uint32, newAesKey string) error {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	stmtUpd, err := tboxDB.Prepare("UPDATE tboxfactorydata_tbl SET bid=?,tboxaes128key=?,eventcreationtime=? where id=?")
	if err != nil {
		logger.Error(err)
		return err
	}
	defer stmtUpd.Close()

	_, err = stmtUpd.Exec(int(bid), newAesKey, eventCreationTime, 1)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func GetEventCreationTime(id int) (uint32, error) {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return 0, err
	}

	var eventCreatTime uint32
	err = tboxDB.QueryRow("SELECT eventcreationtime FROM tboxfactorydata_tbl WHERE id = ?", id).Scan(&eventCreatTime)
	if err != nil {
		logger.Error(err)
		return 0, err
	}

	return eventCreatTime, nil
}

func (ac *Account) getServiceData() ([]byte, error) {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	var tboxserialno, pretboxaes128key, vin, iccid, imsi, tboxaes128key string
	var id, status, bid, eventCreatTime int
	err = tboxDB.QueryRow("SELECT * FROM tboxfactorydata_tbl WHERE id = ?", 1).Scan(
		&id,
		&tboxserialno,
		&pretboxaes128key,
		&vin,
		&iccid,
		&imsi,
		&status,
		&bid,
		&tboxaes128key,
		&eventCreatTime)

	if err != nil {
		logger.Error(err)
		return nil, err
	}

	registerMessageMap := make(map[string]interface{})
	registerMessageMap["pretboxaes128key"] = pretboxaes128key
	registerMessageMap["vin"] = vin
	registerMessageMap["tboxserialno"] = tboxserialno
	registerMessageMap["imsi"] = imsi
	registerMessageMap["rollnumber"] = util.GenRandomString(16)
	registerMessageMap["iccid"] = iccid

	serviceData, err := json.Marshal(registerMessageMap)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	logger.Debug("serviceData =", serviceData)
	logger.Debug("serviceDataJson =", string(serviceData))

	return serviceData, nil
}

func (ac *Account) getDispatchData(encryptServData []byte) ([]byte, error) {
	var dd message.DispatchData
	dd.EventCreationTime = uint32(time.Now().Unix())
	dd.Aid = 0x1
	dd.Mid = 0x1
	dd.MessageCounter = 0
	dd.ServiceDataLength = uint16(len(encryptServData))
	dd.Result = 0
	dd.SecurityVersion = message.Encrypt_No
	dd.DispatchCreationTime = uint32(time.Now().Unix())

	dispatchData, err := util.StructToByteSlice(dd)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return dispatchData, nil
}

func (ac *Account) getMessageHeaderData(serviceData []byte) ([]byte, error) {
	var mh message.MessageHeader
	mh.FixHeader = message.MessageHeaderID
	mh.ServiceDataCheck = util.DataXOR(serviceData)
	mh.ServiceVersion = 0x0 //not sure
	mh.Bid = 0x0
	mh.MessageFlag = 0x0

	messageHeaderData, err := util.StructToByteSlice(mh)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return messageHeaderData, nil
}

func (ac *Account) RegisterReq(eve *Event) error {
	if ac.registerStart {
		logger.Error("Register already started.")
		return errors.New("Register already started.")
	}

	ac.registerStart = true
	if ac.closeRegAckTimer == nil {
		ac.closeRegAckTimer = make(chan bool, 1)
	}

	msg := message.Message{
		Connection: eve.Conn,
	}

	serviceData, err := ac.getServiceData()
	if err != nil {
		logger.Error(err)
		return err
	}

	encryptServData := serviceData

	dispatchData, err := ac.getDispatchData(encryptServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	messageHeaderData, err := ac.getMessageHeaderData(serviceData)
	if err != nil {
		logger.Error(err)
		return err
	}

	data, err := msg.GetOneMessage(messageHeaderData, dispatchData, encryptServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	err = msg.SendOneMessage(data)
	if err != nil {
		logger.Error(err)
		return err
	}

	if ac.regReqTimes < RegisterReqMaxTimes {
		ac.checkRegAckTimer = time.NewTimer(CheckRegAckTimerTime)
		ac.regReqTimes++

		go func() {
			select {
			case <-ac.checkRegAckTimer.C:
				logger.Info("Timer is coming, register again")
				eve.PushEventChannel(RegisterReqEventMessage, nil)
				ac.registerStart = false
			case <-ac.closeRegAckTimer:
				logger.Info("Close RegAckTimer!")
			}
		}()
	} else {
		ac.registerStart = false
		ac.regReqTimes = 0
	}

	return nil
}

func (ac *Account) RegisterACK(eve *Event, msg *message.Message) error {
	if ac.regReqTimes > 0 {
		ac.checkRegAckTimer.Stop()
		ac.closeRegAckTimer <- true
		ac.regReqTimes = 0
	}

	if eve.TboxStatus != TboxUnRegister {
		logger.Error("Already Registered!")
		return errors.New("Already Registered!")
	}

	registerAckMsg := RegisterAckMsg{}

	eventCreatTime, err := GetEventCreationTime(1)
	if err != nil {
		logger.Error(err)
		goto FAILURE
	}

	if eventCreatTime == msg.DisPatch.EventCreationTime {
		logger.Error("Repeated RegisterACK!")
		goto FAILURE
	}

	err = json.Unmarshal(msg.ServData, &registerAckMsg)
	if err != nil {
		logger.Error(err)
		goto FAILURE
	}

	logger.Debug("mh =", msg.MesHeader)
	logger.Debug("dd =", msg.DisPatch)
	logger.Debug("service =", string(msg.ServData))

	//Store DB
	if registerAckMsg.Status == 1 && msg.DisPatch.Result != AlreadyRegister {
		goto FAILURE
	} else {
		err = ac.saveRegisterDataToDB(registerAckMsg.Bid, msg.DisPatch.EventCreationTime, registerAckMsg.CallbackNum)
		if err != nil {
			logger.Error(err)
			goto FAILURE
		}
	}

	eve.TboxStatus = TboxRegisteredUnLogin
	eve.SetTboxStatusToDB(TboxRegisteredUnLogin)
	eve.PushEventChannel(EventLoginRequest, nil)
	ac.registerStart = false
	return nil

FAILURE:
	t := time.NewTimer(CheckRegAgainTimerTime)

	go func() {
		select {
		case <-t.C:
			logger.Info("Register fail, register again!")
			eve.PushEventChannel(RegisterReqEventMessage, nil)
		}
	}()

	ac.registerStart = false

	return err
}
