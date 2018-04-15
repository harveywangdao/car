package thing

import (
	"encoding/json"
	"errors"
	"github.com/harveywangdao/road/log/logger"
	"github.com/harveywangdao/road/message"
	"github.com/harveywangdao/road/util"
	"time"
)

const (
	ReadConfigTimeoutTime = 5 * time.Second

	ReadConfigMessageFlag = 0x0

	ReadConfigReqAid = 0x5
	ReadConfigReqMid = 0x1

	ReadConfigAckAid = 0x5
	ReadConfigAckMid = 0x2
)

const (
	ReadConfigStop = iota
	ReadConfigReqStatus
)

const (
	ThingSN  = "thingsn"
	Version  = "version"
	WorkAddr = "workaddr"
	WorkPort = "workport"
)

type ReadConfig struct {
	readConfTimeoutTimer *time.Timer
	closeTimeoutTimer    chan bool

	readConfigStatus int
}

type ReadConfigReqServData struct {
	IndexList []string `json:"indexlist"`
}

type ReadConfigAckServData struct {
	WorkConfigList []string `json:"workconfiglist"`
}

func (readConf *ReadConfig) GetConfig(indexs []string) []string {
	configData := make([]string, 0, 16)

	for _, index := range indexs {
		switch index {
		case ThingSN:
			configData = append(configData, "12345678901234567890123456711")
		case Version:
			configData = append(configData, "1.0.2")
		case WorkAddr:
			configData = append(configData, "192.168.162.120")
		case WorkPort:
			configData = append(configData, "5525")
		}
	}

	logger.Info("Config:", configData)

	return configData
}

func (readConf *ReadConfig) ReadConfigReq(thing *Thing, reqMsg *message.Message) error {
	if readConf.readConfigStatus != ReadConfigStop {
		logger.Error("ReadConfig already start!")
		return errors.New("ReadConfig already start!")
	}

	if thing.ThingStatus != ThingRegisteredLogined {
		logger.Error("Not login or register")
		return errors.New("Not login or register")
	}

	readConf.readConfigStatus = ReadConfigReqStatus

	thing.PushEventChannel(EventReadConfigAck, reqMsg)

	return nil
}

func (readConf *ReadConfig) ReadConfigAck(thing *Thing, reqMsg *message.Message) error {
	if readConf.readConfigStatus != ReadConfigReqStatus {
		logger.Error("Need ReadConfigReq!")
		return errors.New("Need ReadConfigReq!")
	}

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	readConfReqServData := &ReadConfigReqServData{}
	err := json.Unmarshal(reqMsg.ServData, readConfReqServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("readConfReqServData =", string(reqMsg.ServData))

	readConfAckServData := ReadConfigAckServData{
		WorkConfigList: readConf.GetConfig(readConfReqServData.IndexList),
	}

	serviceData, err := json.Marshal(&readConfAckServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("serviceData =", serviceData)
	logger.Debug("serviceDataJson =", string(serviceData))

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
		EventCreationTime:    reqMsg.DisPatch.EventCreationTime,
		Aid:                  ReadConfigAckAid,
		Mid:                  ReadConfigAckMid,
		MessageCounter:       reqMsg.DisPatch.MessageCounter + 1,
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
		Bid:              reqMsg.MesHeader.Bid,
		MessageFlag:      ReadConfigMessageFlag,
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

	logger.Debug("Send ReadConfigAck Success---")

	readConf.readConfigStatus = ReadConfigStop

	return nil
}
