package tbox

import (
	"encoding/json"
	"errors"
	"hcxy/iov/log/logger"
	"hcxy/iov/message"
	"hcxy/iov/util"
	"time"
)

const (
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
	reloginTimer      *time.Timer
	closeReloginTimer chan bool

	reloginStatus int
}

type ReLoginReqServData struct {
	NewTime byte `json:"newtime"` //NewTime*10 minute
}

func (relogin *ReLogin) ReLoginReq(eve *Event, reqMsg *message.Message) error {
	if relogin.reloginStatus != ReLoginStop {
		logger.Error("Relogin already start!")
		return errors.New("Relogin already start!")
	}

	relogin.reloginStatus = ReLoginReqStatus

	if eve.TboxStatus != TboxRegisteredLogined {
		logger.Error("Not login or register")
		return errors.New("Not login or register")
	}

	reLoginReqServData := &ReLoginReqServData{}
	err := json.Unmarshal(reqMsg.ServData, reLoginReqServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("reLoginReqServData =", reLoginReqServData)

	relogin.reloginTimer = time.NewTimer(time.Duration(reLoginReqServData.NewTime) * 10 * time.Minute)

	if relogin.closeReloginTimer == nil {
		relogin.closeReloginTimer = make(chan bool, 1)
	}

	go func() {
		logger.Debug("Start timer......")
		select {
		case <-relogin.reloginTimer.C:
			logger.Info("Timer coming, need relogin!")
			eve.PushEventChannel(EventLoginRequest, nil)
		case <-relogin.closeReloginTimer:
			logger.Info("Close relogin timer!")
		}

		logger.Info("Timer Close......")
	}()

	eve.PushEventChannel(EventReLoginAck, reqMsg)

	return nil
}

func (relogin *ReLogin) ReLoginAck(eve *Event, reqMsg *message.Message) error {
	if relogin.reloginStatus != ReLoginReqStatus {
		logger.Error("Need ReLoginReq!")
		return errors.New("Need ReLoginReq!")
	}

	msg := message.Message{
		Connection: eve.Conn,
	}

	//Service data
	serviceData := make([]byte, 1)

	logger.Debug("serviceData =", serviceData)

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
		EventCreationTime:    reqMsg.DisPatch.EventCreationTime,
		Aid:                  ReLoginAckAid,
		Mid:                  ReLoginAckMid,
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

	relogin.reloginStatus = ReLoginStop

	return nil
}