package thing

import (
	"encoding/json"
	"errors"
	"github.com/harveywangdao/road/database"
	"github.com/harveywangdao/road/log/logger"
	"github.com/harveywangdao/road/message"
	"github.com/harveywangdao/road/util"
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

type Register struct {
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

func (reg *Register) saveRegisterDataToDB(bid, eventCreationTime uint32, newAesKey string) error {
	thingDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	stmtUpd, err := thingDB.Prepare("UPDATE thingbaseinfodata_tbl SET bid=?,thingaes128key=?,eventcreationtime=? where id=?")
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
	thingDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return 0, err
	}

	var eventCreatTime uint32
	err = thingDB.QueryRow("SELECT eventcreationtime FROM thingbaseinfodata_tbl WHERE id = ?", id).Scan(&eventCreatTime)
	if err != nil {
		logger.Error(err)
		return 0, err
	}

	return eventCreatTime, nil
}

func (reg *Register) getServiceData() ([]byte, error) {
	thingDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	var thingserialno, prethingaes128key, thingid, iccid, imsi, thingaes128key string
	var id, status, bid, eventCreatTime int
	err = thingDB.QueryRow("SELECT * FROM thingbaseinfodata_tbl WHERE id = ?", 1).Scan(
		&id,
		&thingserialno,
		&prethingaes128key,
		&thingid,
		&iccid,
		&imsi,
		&status,
		&bid,
		&thingaes128key,
		&eventCreatTime)

	if err != nil {
		logger.Error(err)
		return nil, err
	}

	registerMessageMap := make(map[string]interface{})
	registerMessageMap["prethingaes128key"] = prethingaes128key
	registerMessageMap["thingid"] = thingid
	registerMessageMap["thingserialno"] = thingserialno
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

func (reg *Register) getDispatchData(encryptServData []byte) ([]byte, error) {
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

func (reg *Register) getMessageHeaderData(serviceData []byte) ([]byte, error) {
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

func (reg *Register) RegisterReq(thing *Thing) error {
	if reg.registerStart {
		logger.Error("Register already started.")
		return errors.New("Register already started.")
	}

	reg.registerStart = true
	if reg.closeRegAckTimer == nil {
		reg.closeRegAckTimer = make(chan bool, 1)
	}

	msg := message.Message{
		Connection: thing.Conn,
	}

	serviceData, err := reg.getServiceData()
	if err != nil {
		logger.Error(err)
		return err
	}

	encryptServData := serviceData

	dispatchData, err := reg.getDispatchData(encryptServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	messageHeaderData, err := reg.getMessageHeaderData(serviceData)
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

	if reg.regReqTimes < RegisterReqMaxTimes {
		reg.checkRegAckTimer = time.NewTimer(CheckRegAckTimerTime)
		reg.regReqTimes++

		go func() {
			select {
			case <-reg.checkRegAckTimer.C:
				logger.Warn("Timer is coming, register again")
				thing.PushEventChannel(RegisterReqEventMessage, nil)
				reg.registerStart = false
			case <-reg.closeRegAckTimer:
				logger.Debug("Close RegAckTimer!")
			}
		}()
	} else {
		reg.registerStart = false
		reg.regReqTimes = 0
	}

	return nil
}

func (reg *Register) RegisterACK(thing *Thing, msg *message.Message) error {
	if reg.regReqTimes > 0 {
		reg.checkRegAckTimer.Stop()
		reg.closeRegAckTimer <- true
		reg.regReqTimes = 0
	}

	if thing.ThingStatus != ThingUnRegister {
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
		logger.Error("Register fail!")
		goto FAILURE
	} else {
		err = reg.saveRegisterDataToDB(registerAckMsg.Bid, msg.DisPatch.EventCreationTime, registerAckMsg.CallbackNum)
		if err != nil {
			logger.Error(err)
			goto FAILURE
		}
	}

	thing.ThingStatus = ThingRegisteredUnLogin
	thing.SetThingStatusToDB(ThingRegisteredUnLogin)
	thing.PushEventChannel(EventLoginRequest, nil)
	reg.registerStart = false
	return nil

FAILURE:
	t := time.NewTimer(CheckRegAgainTimerTime)

	go func() {
		select {
		case <-t.C:
			logger.Info("Register fail, register again!")
			thing.PushEventChannel(RegisterReqEventMessage, nil)
		}
	}()

	reg.registerStart = false

	return err
}
