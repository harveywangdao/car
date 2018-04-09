package tboxservice

import (
	"encoding/json"
	"errors"
	"hcxy/iov/log/logger"
	"hcxy/iov/message"
	"hcxy/iov/util"
	"time"
)

const (
	ReloginTimeoutTime = 2 * time.Second

	ReLoginMessageFlag = 0x0

	ReLoginReqAid = 0x4
	ReLoginReqMid = 0x1

	ReLoginAckAid = 0x4
	ReLoginAckMid = 0x2
)

const (
	ReLoginStop = iota
	ReLoginReqStatus
	/*ReLoginAckStatus*/
)

type ReLogin struct {
	timeoutTimer      *time.Timer
	closeTimeoutTimer chan bool

	reloginStatus int
}

type ReLoginReqServData struct {
	NewTime byte `json:"newtime"` //NewTime*10 minute
}

func (relogin *ReLogin) ReLoginReq(eve *Event) error {
	if relogin.reloginStatus != ReLoginStop {
		logger.Error("Relogin already start!")
		return errors.New("Relogin already start!")
	}

	/*	if eve.TboxStatus != TboxRegisteredLogined{
		logger.Error("Not login or register")
		return errors.New("Not login or register")
	}*/

	relogin.reloginStatus = ReLoginReqStatus

	msg := message.Message{
		Connection: eve.Conn,
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

	aesKey, err := eve.GetAesKey()
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

	//Message header data
	mh := message.MessageHeader{
		FixHeader:        message.MessageHeaderID,
		ServiceDataCheck: util.DataXOR(serviceData),
		ServiceVersion:   0x0, //not sure
		Bid:              eve.GetBid(),
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

	logger.Debug("Send ReLoginReq Success---")

	relogin.timeoutTimer = time.NewTimer(ReloginTimeoutTime)

	if relogin.closeTimeoutTimer == nil {
		relogin.closeTimeoutTimer = make(chan bool, 1)
	}

	go func() {
		logger.Info("Start timer......")
		select {
		case <-relogin.timeoutTimer.C:
			logger.Info("Timeout timer coming, relogin fail!")
			eve.PushEventChannel(EventReLoginRequest, nil)
			relogin.reloginStatus = ReLoginStop
		case <-relogin.closeTimeoutTimer:
			logger.Info("Close Timeout timer!")
		}

		logger.Info("Timer Close......")
	}()

	return nil
}

func (relogin *ReLogin) ReLoginAck(eve *Event, respMsg *message.Message) error {
	if relogin.reloginStatus != ReLoginReqStatus {
		logger.Error("Need ReLoginReq!")
		return errors.New("Need ReLoginReq!")
	}

	relogin.timeoutTimer.Stop()
	relogin.closeTimeoutTimer <- true

	relogin.reloginStatus = ReLoginStop

	return nil
}
