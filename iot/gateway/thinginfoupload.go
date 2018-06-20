package gateway

import (
	"encoding/json"
	"errors"
	"github.com/harveywangdao/road/database/mongo"
	"github.com/harveywangdao/road/log/logger"
	"github.com/harveywangdao/road/message"
	"github.com/harveywangdao/road/util"
	"time"
)

const (
	ThingInfoUploadTimeoutTime = 5 * time.Second
	ThingInfoUploadMessageFlag = 0x1

	ThingInfoUploadAid = 0xF5
	ThingInfoUploadMid = 0x4

	ThingInfoUploadAckAid = 0xF5
	ThingInfoUploadAckMid = 0x2
)

const (
	ThingInfoUploadStop = iota
	ThingInfoUploadStatus
)

type ThingInfoUpload struct {
	thingInfoUploadTimer      *time.Timer
	closeThingInfoUploadTimer chan bool

	thingInfoUploadStatus int
}

func (upload *ThingInfoUpload) ThingInfoUploadReq(thing *Thing, reqMsg *message.Message) error {
	if upload.thingInfoUploadStatus != ThingInfoUploadStop {
		logger.Error("ThingInfoUpload already start!")
		return errors.New("ThingInfoUpload already start!")
	}

	upload.thingInfoUploadStatus = ThingInfoUploadStatus

	thingInfor := &ThingInfor{}
	err := json.Unmarshal(reqMsg.ServData, thingInfor)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("thingInfor =", string(reqMsg.ServData))

	session, err := mongo.CloneMgoSession()
	if err != nil {
		logger.Error(err)
		return err
	}
	defer session.Close()

	c := session.DB("iotmgodb").C("ThingInforData")
	err = c.Insert(thingInfor)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("Save to mongo!")

	thing.PushEventChannel(EventThingInfoUploadAck, reqMsg)

	return nil
}

func (upload *ThingInfoUpload) ThingInfoUploadAck(thing *Thing, reqMsg *message.Message) error {
	if upload.thingInfoUploadStatus != ThingInfoUploadStatus {
		logger.Error("Need ThingInfoUpload!")
		return errors.New("Need ThingInfoUpload!")
	}

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	serviceData := make([]byte, 1)
	logger.Debug("serviceData =", serviceData)

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
		Aid:                  ThingInfoUploadAckAid,
		Mid:                  ThingInfoUploadAckMid,
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
		MessageFlag:      ThingInfoUploadMessageFlag,
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

	logger.Debug("Send ThingInfoUploadAck Success---")

	upload.thingInfoUploadStatus = ThingInfoUploadStop

	return nil
}
