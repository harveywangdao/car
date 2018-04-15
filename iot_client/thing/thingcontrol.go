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
	ThingControlTimeoutTime = 2 * time.Second

	ThingControlMessageFlag = 0x1

	RemoteOperationRequestAid = 0xF1
	RemoteOperationRequestMid = 0x1

	DispatcherAckMessageAid = 0xF1
	DispatcherAckMessageMid = 0x2

	RemoteOperationEndAid = 0xF1
	RemoteOperationEndMid = 0x3

	RemoteOperationAckAid = 0xF1
	RemoteOperationAckMid = 0x4

	RemoteOperationSuccess = 1
)

const (
	ThingControlStop = iota
	RemoteOperationReqStatus
	DispatcherAckMessageStatus
	RemoteOperationEndStatus
	RemoteOperationAckStatus
)

type ThingControl struct {
	timeoutTimer      *time.Timer
	closeTimeoutTimer chan bool

	thingControlStatus int
	operation          uint16
}

type RemoteOperationReqServData struct {
	Operation          uint16 `json:"operation"`
	OperationParameter int64  `json:"operationparameter"`
}

type DispatcherAckMessageServData struct {
	Operation uint16 `json:"operation"`
}

type RemoteOperationEndServData struct {
	Operation uint16 `json:"operation"`
	Parameter int64  `json:"parameter"`
}

type RemoteOperationAckServData struct {
	Operation uint16 `json:"operation"`
	Status    byte   `json:"status"`
	Parameter int64  `json:"parameter"`
}

const (
	CentralLockOpen     = 0x00F1
	CentralLockClose    = 0x00F2
	WindowClose         = 0x00F4
	WhistleAndFlash     = 0x00F5
	AirConditionerOpen  = 0x00F6
	AirConditionerClose = 0x00F7
	EngineStart         = 0x00F8
	EngineStop          = 0x00F9
	SkyWindowOpen       = 0x00FA
	SkyWindowClose      = 0x00FB
	FrontDefrostStart   = 0x0001
	FrontDefrostStop    = 0x0002
	BackDefrostStart    = 0x0003
	BackDefrostStop     = 0x0004
	SeatheatStart       = 0x0005
	SeatheatStop        = 0x0006
	TwoFlashStart       = 0x0007
	ThingDefence        = 0x0008
	ThingUndefence      = 0x0009
	EngineLock          = 0x000A
	EngineUnlock        = 0x000B

	UnknownRemoteOparetion = 0xFFFF
)

func (vc *ThingControl) convertOperation(op string) (uint16, error) {
	switch op {
	case "lock":
		return EngineLock, nil
	case "unlock":
		return EngineUnlock, nil
	case "defence":
		return ThingDefence, nil
	case "undefence":
		return ThingUndefence, nil
	default:
		logger.Error("Unknown remote operation!")
	}

	return UnknownRemoteOparetion, errors.New("Unknown remote operation!")
}

func (vc *ThingControl) RemoteOperationReq(thing *Thing, reqMsg *message.Message) error {
	if vc.thingControlStatus != ThingControlStop {
		logger.Error("ThingControl already start!")
		return errors.New("ThingControl already start!")
	}

	vc.thingControlStatus = RemoteOperationReqStatus

	remoteOperationReqServData := &RemoteOperationReqServData{}
	err := json.Unmarshal(reqMsg.ServData, remoteOperationReqServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	vc.operation = remoteOperationReqServData.Operation

	logger.Debug("remoteOperationReqServData =", string(reqMsg.ServData))

	//thing.PushEventChannel(EventDispatcherAckMessage, reqMsg)
	err = vc.dispatcherAckMessage1(thing, reqMsg)
	if err != nil {
		logger.Error(err)
		return err
	}

	thing.PushEventChannel(EventRemoteOperationEnd, reqMsg)

	return nil
}

func (vc *ThingControl) dispatcherAckMessage1(thing *Thing, reqMsg *message.Message) error {
	if vc.thingControlStatus != RemoteOperationReqStatus {
		logger.Error("Need RemoteOperationReqStatus!")
		return errors.New("Need RemoteOperationReqStatus!")
	}

	vc.thingControlStatus = DispatcherAckMessageStatus

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	dispatcherAckMessageServData := DispatcherAckMessageServData{
		Operation: vc.operation,
	}

	serviceData, err := json.Marshal(&dispatcherAckMessageServData)
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
		Aid:                  DispatcherAckMessageAid,
		Mid:                  DispatcherAckMessageMid,
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
		MessageFlag:      ThingControlMessageFlag,
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

	logger.Debug("Send DispatcherAckMessage Success---")

	return nil
}

func (vc *ThingControl) RemoteOperationEnd(thing *Thing, reqMsg *message.Message) error {
	if vc.thingControlStatus != DispatcherAckMessageStatus {
		logger.Error("Need DispatcherAckMessageStatus!")
		return errors.New("Need DispatcherAckMessageStatus!")
	}

	vc.thingControlStatus = RemoteOperationEndStatus

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	remoteOperationEndServData := RemoteOperationEndServData{
		Operation: vc.operation,
		Parameter: 0, //need fix
	}

	serviceData, err := json.Marshal(&remoteOperationEndServData)
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
		Aid:                  RemoteOperationEndAid,
		Mid:                  RemoteOperationEndMid,
		MessageCounter:       reqMsg.DisPatch.MessageCounter + 2,
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
		MessageFlag:      ThingControlMessageFlag,
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

	logger.Debug("Send RemoteOperationEnd Success---")

	vc.timeoutTimer = time.NewTimer(ThingControlTimeoutTime)

	if vc.closeTimeoutTimer == nil {
		vc.closeTimeoutTimer = make(chan bool, 1)
	}

	go func() {
		logger.Debug("Start timer......")
		select {
		case <-vc.timeoutTimer.C:
			logger.Error("Timeout timer coming, vc fail!")
			//thing.PushEventChannel(EventThingControlRequest, nil)
			vc.thingControlStatus = ThingControlStop
		case <-vc.closeTimeoutTimer:
			logger.Debug("Close Timeout timer!")
		}

		logger.Debug("Timer Close......")
	}()

	return nil
}

func (vc *ThingControl) DispatcherAckMessage2(thing *Thing, ackMsg *message.Message) error {
	if vc.thingControlStatus == RemoteOperationEndStatus {
		vc.timeoutTimer.Stop()

		//vc.thingControlStatus = DispatcherAckMessageStatus

		dispatcherAckMessageServData := &DispatcherAckMessageServData{}
		err := json.Unmarshal(ackMsg.ServData, dispatcherAckMessageServData)
		if err != nil {
			logger.Error(err)
			return err
		}

		logger.Debug("dispatcherAckMessageServData =", string(ackMsg.ServData))

		if dispatcherAckMessageServData.Operation != vc.operation {
			vc.closeTimeoutTimer <- true
			vc.thingControlStatus = ThingControlStop

			return errors.New("operation not right!")
		}

		thing.PushEventChannel(EventRemoteOperationAck, ackMsg)
	} else if vc.thingControlStatus == RemoteOperationAckStatus {
		vc.timeoutTimer.Stop()

		dispatcherAckMessageServData := &DispatcherAckMessageServData{}
		err := json.Unmarshal(ackMsg.ServData, dispatcherAckMessageServData)
		if err != nil {
			logger.Error(err)
			return err
		}

		logger.Debug("dispatcherAckMessageServData =", string(ackMsg.ServData))

		vc.closeTimeoutTimer <- true
		vc.thingControlStatus = ThingControlStop
	} else {
		logger.Error("Need RemoteOperationEndStatus!")
		return errors.New("Need RemoteOperationEndStatus!")
	}

	return nil
}

func (vc *ThingControl) RemoteOperationAck(thing *Thing, ackMsg *message.Message) error {
	if vc.thingControlStatus != RemoteOperationEndStatus {
		logger.Error("Need RemoteOperationEndStatus!")
		return errors.New("Need RemoteOperationEndStatus!")
	}

	vc.thingControlStatus = RemoteOperationAckStatus

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	remoteOperationAckServData := RemoteOperationAckServData{
		Operation: vc.operation,
		Status:    RemoteOperationSuccess,
		Parameter: 0,
	}

	serviceData, err := json.Marshal(&remoteOperationAckServData)
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
		EventCreationTime:    ackMsg.DisPatch.EventCreationTime,
		Aid:                  RemoteOperationAckAid,
		Mid:                  RemoteOperationAckMid,
		MessageCounter:       ackMsg.DisPatch.MessageCounter + 1,
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
		Bid:              ackMsg.MesHeader.Bid,
		MessageFlag:      ThingControlMessageFlag,
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

	logger.Debug("Send RemoteOperationAck Success---")

	vc.timeoutTimer.Reset(ThingControlTimeoutTime)

	return nil
}
