package gateway

import (
	"encoding/json"
	"errors"
	/*"gopkg.in/mgo.v2/bson"*/
	//"github.com/harveywangdao/road/database"
	//"github.com/harveywangdao/road/database/mongo"
	"github.com/harveywangdao/road/log/logger"
	"github.com/harveywangdao/road/message"
	"github.com/harveywangdao/road/util"
	//"github.com/jinzhu/gorm"
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

	readConfReqServData *ReadConfigReqServData
}

type ReadConfigReqServData struct {
	IndexList []string `json:"indexlist"`
}

type ReadConfigAckServData struct {
	WorkConfigList []string `json:"workconfiglist"`
}

func (readConf *ReadConfig) GetIndexList() []string {
	indexs := make([]string, 0, 16)
	indexs = append(indexs, ThingSN)
	indexs = append(indexs, Version)
	indexs = append(indexs, WorkPort)
	return indexs
}

func (readConf *ReadConfig) ReadConfigReq(thing *Thing) error {
	if readConf.readConfigStatus != ReadConfigStop {
		logger.Error("ReadConfig already start!")
		return errors.New("ReadConfig already start!")
	}

	/*	if thing.ThingStatus != ThingRegisteredLogined{
		logger.Error("Not login or register")
		return errors.New("Not login or register")
	}*/

	readConf.readConfigStatus = ReadConfigReqStatus

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	readConf.readConfReqServData = &ReadConfigReqServData{
		IndexList: readConf.GetIndexList(),
	}

	serviceData, err := json.Marshal(readConf.readConfReqServData)
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
		Aid:                  ReadConfigReqAid,
		Mid:                  ReadConfigReqMid,
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

	logger.Debug("Send ReadConfigReq Success---")

	readConf.readConfTimeoutTimer = time.NewTimer(ReadConfigTimeoutTime)

	if readConf.closeTimeoutTimer == nil {
		readConf.closeTimeoutTimer = make(chan bool, 1)
	}

	go func() {
		logger.Debug("Start timer......")
		select {
		case <-readConf.readConfTimeoutTimer.C:
			logger.Warn("Timeout timer coming, readConf fail!")
			//thing.PushEventChannel(EventReadConfigRequest, nil)
			readConf.readConfigStatus = ReadConfigStop
		case <-readConf.closeTimeoutTimer:
			logger.Debug("Close Timeout timer!")
		}

		logger.Debug("Timer Close......")
	}()

	return nil
}

func (readConf *ReadConfig) ReadConfigAck(thing *Thing, ackMsg *message.Message) error {
	if readConf.readConfigStatus != ReadConfigReqStatus {
		logger.Error("Need ReadConfigReq!")
		return errors.New("Need ReadConfigReq!")
	}

	readConf.readConfTimeoutTimer.Stop()
	readConf.closeTimeoutTimer <- true

	readConfAckServData := &ReadConfigAckServData{}
	err := json.Unmarshal(ackMsg.ServData, readConfAckServData)
	if err != nil {
		logger.Error(err)
		readConf.readConfigStatus = ReadConfigStop
		return err
	}

	logger.Info("readConfAckServData =", string(ackMsg.ServData))
	logger.Debug("WorkConfigList =", readConfAckServData.WorkConfigList)

	readConf.readConfigStatus = ReadConfigStop

	return nil
}
