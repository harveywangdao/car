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
	HeartbeatTimeoutTime = 5 * time.Second
	HeartbeatMessageFlag = 0x0

	HeartbeatReqAid = 0xB
	HeartbeatReqMid = 0x1

	HeartbeatAckAid = 0xB
	HeartbeatAckMid = 0x2
)

const (
	HeartbeatStop = iota
	HeartbeatReqStatus
	/*HeartbeatAckStatus*/
)

type Heartbeat struct {
	heartbeatTimer      *time.Timer
	closeHeartbeatTimer chan bool

	heartbeatStatus int
}

type HeartbeatReqServData struct {
	AppointMF  byte `json:"appointmf"`
	AppointAid byte `json:"appointaid"`
}

func (hb *Heartbeat) HeartbeatReq(thing *Thing) error {
	if hb.heartbeatStatus != HeartbeatStop {
		logger.Error("Heartbeat already start!")
		return errors.New("Heartbeat already start!")
	}

	hb.heartbeatStatus = HeartbeatReqStatus

	if thing.ThingStatus != ThingRegisteredLogined {
		logger.Error("Not login or register")
		return errors.New("Not login or register")
	}

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	heartbeatReqServData := HeartbeatReqServData{
		AppointMF:  255,
		AppointAid: 0,
	}

	serviceData, err := json.Marshal(&heartbeatReqServData)
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
		EventCreationTime:    uint32(time.Now().Unix()),
		Aid:                  HeartbeatReqAid,
		Mid:                  HeartbeatReqMid,
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
		MessageFlag:      HeartbeatMessageFlag,
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

	logger.Debug("Send HeartbeatReq Success---")

	hb.heartbeatTimer = time.NewTimer(HeartbeatTimeoutTime)

	if hb.closeHeartbeatTimer == nil {
		hb.closeHeartbeatTimer = make(chan bool, 1)
	}

	go func() {
		logger.Debug("Start timer......")
		select {
		case <-hb.heartbeatTimer.C:
			logger.Warn("Timer coming, need Heartbeat again!")
			hb.heartbeatStatus = HeartbeatStop
			thing.PushEventChannel(EventHeartbeatRequest, nil)
		case <-hb.closeHeartbeatTimer:
			logger.Debug("Close Heartbeat timer!")
		}

		logger.Debug("Timer Close......")
	}()

	return nil
}

func (hb *Heartbeat) HeartbeatAck(thing *Thing, reqMsg *message.Message) error {
	if hb.heartbeatStatus != HeartbeatReqStatus {
		logger.Error("Need HeartbeatReq!")
		return errors.New("Need HeartbeatReq!")
	}

	hb.heartbeatTimer.Stop()
	hb.closeHeartbeatTimer <- true

	hb.heartbeatStatus = HeartbeatStop

	return nil
}
