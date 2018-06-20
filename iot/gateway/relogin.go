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
	ReloginTimeoutTime = 10 * time.Second
	ReLoginAgainTime   = 10 * time.Second

	ReLoginMessageFlag = 0x0

	ReLoginReqAid = 0x4
	ReLoginReqMid = 0x1

	ReLoginAckAid = 0x4
	ReLoginAckMid = 0x2
)

const (
	ReLoginStop = iota
	ReLoginReqStatus
)

type ReLogin struct {
	timeoutTimer      *time.Timer
	closeTimeoutTimer chan bool

	reloginStatus         int
	reLoginEventCreatTime uint32
}

type ReLoginReqServData struct {
	NewTime byte `json:"newtime"` //NewTime*10 minute
}

func (relogin *ReLogin) reLoginReqSendData(thing *Thing) error {
	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	reLoginReqServData := ReLoginReqServData{
		NewTime: 1,
	}

	serviceData, err := json.Marshal(&reLoginReqServData)
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
		Aid:                  ReLoginReqAid,
		Mid:                  ReLoginReqMid,
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

	relogin.reLoginEventCreatTime = dd.EventCreationTime

	//Message header data
	mh := message.MessageHeader{
		FixHeader:        message.MessageHeaderID,
		ServiceDataCheck: util.DataXOR(serviceData),
		ServiceVersion:   0x0, //not sure
		Bid:              thing.GetBid(),
		MessageFlag:      ReLoginMessageFlag,
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

	return nil
}

func (relogin *ReLogin) ReLoginReq(thing *Thing) error {
	if relogin.reloginStatus != ReLoginStop {
		logger.Error("Relogin already start!")
		return errors.New("Relogin already start!")
	}

	/*	if thing.TboxStatus != TboxRegisteredLogined{
		logger.Error("Not login or register")
		return errors.New("Not login or register")
	}*/

	relogin.reloginStatus = ReLoginReqStatus

	err := relogin.reLoginReqSendData(thing)
	if err != nil {
		t := time.NewTimer(ReLoginAgainTime)

		go func() {
			select {
			case <-t.C:
				logger.Info("ReLogin fail, ReLogin again!")
				thing.PushEventChannel(EventReLoginRequest, nil)
			}
		}()

		relogin.reloginStatus = ReLoginStop

		return err
	}

	logger.Debug("Send ReLoginReq Success---")

	relogin.timeoutTimer = time.NewTimer(ReloginTimeoutTime)

	if relogin.closeTimeoutTimer == nil {
		relogin.closeTimeoutTimer = make(chan bool, 1)
	}

	go func() {
		logger.Debug("Start timer......")
		select {
		case <-relogin.timeoutTimer.C:
			logger.Warn("Timeout timer coming, relogin fail!")
			thing.PushEventChannel(EventReLoginRequest, nil)
			relogin.reloginStatus = ReLoginStop
		case <-relogin.closeTimeoutTimer:
			logger.Debug("Close Timeout timer!")
		}

		logger.Debug("Timer Close......")
	}()

	return nil
}

func (relogin *ReLogin) ReLoginAck(thing *Thing, respMsg *message.Message) error {
	if relogin.reloginStatus != ReLoginReqStatus {
		logger.Error("Need ReLoginReq!")
		return errors.New("Need ReLoginReq!")
	}

	if relogin.reLoginEventCreatTime != respMsg.DisPatch.EventCreationTime {
		logger.Error("Package out of date!")
		return errors.New("Package out of date!")
	}

	relogin.timeoutTimer.Stop()
	relogin.closeTimeoutTimer <- true

	relogin.reloginStatus = ReLoginStop

	return nil
}
