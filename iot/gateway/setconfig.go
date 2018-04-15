package gateway

import (
	"encoding/json"
	"errors"
	"github.com/harveywangdao/road/log/logger"
	"github.com/harveywangdao/road/message"
	"github.com/harveywangdao/road/util"
	"time"
)

const (
	SetConfigTimeoutTime = 5 * time.Second

	SetConfigMessageFlag = 0x0

	SetConfigReqAid = 0x7
	SetConfigReqMid = 0x1

	SetConfigAckAid = 0x7
	SetConfigAckMid = 0x2
)

const (
	SetConfigStop = iota
	SetConfigReqStatus
)

/*const (
	ThingSN  = "thingsn"
	Version  = "version"
	WorkAddr = "workaddr"
	WorkPort = "workport"
)*/

type SetConfig struct {
	setConfigTimeoutTimer *time.Timer
	closeTimeoutTimer     chan bool

	setConfigigStatus int

	setConfigReqServData *SetConfigReqServData
}

type SetConfigReqServData struct {
	IndexList      []string `json:"indexlist"`
	WorkConfigList []string `json:"workconfiglist"`
}

type SetConfigAckServData struct {
	IndexList      []string `json:"indexlist"`
	WorkConfigList []string `json:"workconfiglist"`
}

func (setConfig *SetConfig) GetIndexList() []string {
	indexs := make([]string, 0, 16)
	indexs = append(indexs, ThingSN)
	indexs = append(indexs, Version)
	indexs = append(indexs, WorkPort)
	return indexs
}

func (setConfig *SetConfig) GetWorkConfigList() []string {
	Configs := make([]string, 0, 16)
	Configs = append(Configs, "12345678901234567890123456711")
	Configs = append(Configs, "1.0.2")
	Configs = append(Configs, "5525")
	return Configs
}

func (setConfig *SetConfig) SetConfigReq(thing *Thing) error {
	if setConfig.setConfigigStatus != SetConfigStop {
		logger.Error("SetConfig already start!")
		return errors.New("SetConfig already start!")
	}

	/*	if thing.ThingStatus != ThingRegisteredLogined{
		logger.Error("Not login or register")
		return errors.New("Not login or register")
	}*/

	setConfig.setConfigigStatus = SetConfigReqStatus

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	setConfig.setConfigReqServData = &SetConfigReqServData{
		IndexList:      setConfig.GetIndexList(),
		WorkConfigList: setConfig.GetWorkConfigList(),
	}

	serviceData, err := json.Marshal(setConfig.setConfigReqServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("serviceData =", serviceData)
	logger.Info("serviceDataJson =", string(serviceData))

	aesKey, err := thing.GetAesKey()
	if err != nil {
		logger.Error(err)
		return err
	}

	encryptServData, err := msg.EncryptServiceData(message.Encrypt_AES128, aesKey, serviceData)
	if err != nil {
		logger.Error(err)
		return err
	}

	//Dispatch data
	dd := message.DispatchData{
		EventCreationTime:    uint32(time.Now().Unix()),
		Aid:                  SetConfigReqAid,
		Mid:                  SetConfigReqMid,
		MessageCounter:       0,
		ServiceDataLength:    uint16(len(encryptServData)),
		Result:               0,
		SecurityVersion:      message.Encrypt_AES128,
		DispatchCreationTime: uint32(time.Now().Unix()),
	}

	dispatchData, err := util.StructToByteSlice(dd)
	if err != nil {
		logger.Error(err)
		return err
	}

	//Message header data
	mh := message.MessageHeader{
		FixHeader:        message.MessageHeaderID,
		ServiceDataCheck: util.DataXOR(serviceData),
		ServiceVersion:   0x0, //not sure
		Bid:              thing.GetBid(),
		MessageFlag:      SetConfigMessageFlag,
	}

	messageHeaderData, err := util.StructToByteSlice(mh)
	if err != nil {
		logger.Error(err)
		return err
	}

	//Send message
	err = msg.SendMessage(messageHeaderData, dispatchData, encryptServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("Send SetConfigReq Success---")

	setConfig.setConfigTimeoutTimer = time.NewTimer(SetConfigTimeoutTime)

	if setConfig.closeTimeoutTimer == nil {
		setConfig.closeTimeoutTimer = make(chan bool, 1)
	}

	go func() {
		logger.Debug("Start timer......")
		select {
		case <-setConfig.setConfigTimeoutTimer.C:
			logger.Warn("Timeout timer coming, setConfig fail!")
			//thing.PushEventChannel(EventSetConfigRequest, nil)
			setConfig.setConfigigStatus = SetConfigStop
		case <-setConfig.closeTimeoutTimer:
			logger.Debug("Close Timeout timer!")
		}

		logger.Debug("Timer Close......")
	}()

	return nil
}

func (setConfig *SetConfig) SetConfigAck(thing *Thing, ackMsg *message.Message) error {
	if setConfig.setConfigigStatus != SetConfigReqStatus {
		logger.Error("Need SetConfigReq!")
		return errors.New("Need SetConfigReq!")
	}

	setConfig.setConfigTimeoutTimer.Stop()
	setConfig.closeTimeoutTimer <- true

	setConfigAckServData := &SetConfigAckServData{}
	err := json.Unmarshal(ackMsg.ServData, setConfigAckServData)
	if err != nil {
		logger.Error(err)
		setConfig.setConfigigStatus = SetConfigStop
		return err
	}

	logger.Info("setConfigAckServData =", string(ackMsg.ServData))
	logger.Debug("WorkConfigList =", setConfigAckServData.WorkConfigList)

	setConfig.setConfigigStatus = SetConfigStop

	return nil
}
