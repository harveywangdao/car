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

func (upload *ThingInfoUpload) ThingInfoUploadReq(thing *Thing) error {
	if upload.thingInfoUploadStatus != ThingInfoUploadStop {
		logger.Error("ThingInfoUpload already start!")
		return errors.New("ThingInfoUpload already start!")
	}

	upload.thingInfoUploadStatus = ThingInfoUploadStatus

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	thingInfor := ThingInfor{}
	thingInfor.GetThingInfor()

	serviceData, err := json.Marshal(&thingInfor)
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
		Aid:                  ThingInfoUploadAid,
		Mid:                  ThingInfoUploadMid,
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

	logger.Debug("Send ThingInfoUpload Success---")

	upload.thingInfoUploadTimer = time.NewTimer(ThingInfoUploadTimeoutTime)

	if upload.closeThingInfoUploadTimer == nil {
		upload.closeThingInfoUploadTimer = make(chan bool, 1)
	}

	go func() {
		logger.Debug("Start timer......")
		select {
		case <-upload.thingInfoUploadTimer.C:
			logger.Warn("Timer coming, need ThingInfoUpload again!")

			upload.thingInfoUploadStatus = ThingInfoUploadStop
			thing.PushEventChannel(EventThingInfoUpload, nil)

		case <-upload.closeThingInfoUploadTimer:
			logger.Debug("Close ThingInfoUpload timer!")
		}

		logger.Debug("Timer Close......")
	}()

	return nil
}

func (upload *ThingInfoUpload) ThingInfoUploadAck(thing *Thing, ackMsg *message.Message) error {
	if upload.thingInfoUploadStatus != ThingInfoUploadStatus {
		logger.Error("Need ThingInfoUpload!")
		return errors.New("Need ThingInfoUpload!")
	}

	upload.thingInfoUploadTimer.Stop()
	upload.closeThingInfoUploadTimer <- true

	upload.thingInfoUploadStatus = ThingInfoUploadStop

	return nil
}
