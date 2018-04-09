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
	VehicleControlTimeoutTime = 2 * time.Second

	VehicleControlMessageFlag = 0x1

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
	VehicleControlStop = iota
	RemoteOperationReqStatus
	DispatcherAckMessageStatus
	RemoteOperationEndStatus
	RemoteOperationAckStatus
)

type VehicleControl struct {
	timeoutTimer      *time.Timer
	closeTimeoutTimer chan bool

	vehicleControlStatus int
	operation            uint16
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
	VehicleDefence      = 0x0008
	VehicleUndefence    = 0x0009
	EngineLock          = 0x000A
	EngineUnlock        = 0x000B

	UnknownRemoteOparetion = 0xFFFF
)

func (vc *VehicleControl) convertOperation(op string) (uint16, error) {
	switch op {
	case "lock":
		return EngineLock, nil
	case "unlock":
		return EngineUnlock, nil
	case "defence":
		return VehicleDefence, nil
	case "undefence":
		return VehicleUndefence, nil
	default:
		logger.Error("Unknown remote operation!")
	}

	return UnknownRemoteOparetion, errors.New("Unknown remote operation!")
}

func (vc *VehicleControl) RemoteOperationReq(eve *Event, reqMsg *message.Message) error {
	if vc.vehicleControlStatus != VehicleControlStop {
		logger.Error("VehicleControl already start!")
		return errors.New("VehicleControl already start!")
	}

	vc.vehicleControlStatus = RemoteOperationReqStatus

	remoteOperationReqServData := &RemoteOperationReqServData{}
	err := json.Unmarshal(reqMsg.ServData, remoteOperationReqServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	vc.operation = remoteOperationReqServData.Operation

	logger.Debug("remoteOperationReqServData =", string(reqMsg.ServData))

	//eve.PushEventChannel(EventDispatcherAckMessage, reqMsg)
	err = vc.dispatcherAckMessage1(eve, reqMsg)
	if err != nil {
		logger.Error(err)
		return err
	}

	eve.PushEventChannel(EventRemoteOperationEnd, reqMsg)

	return nil
}

func (vc *VehicleControl) dispatcherAckMessage1(eve *Event, reqMsg *message.Message) error {
	if vc.vehicleControlStatus != RemoteOperationReqStatus {
		logger.Error("Need RemoteOperationReqStatus!")
		return errors.New("Need RemoteOperationReqStatus!")
	}

	vc.vehicleControlStatus = DispatcherAckMessageStatus

	msg := message.Message{
		Connection: eve.Conn,
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
		MessageFlag:      VehicleControlMessageFlag,
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

func (vc *VehicleControl) RemoteOperationEnd(eve *Event, reqMsg *message.Message) error {
	if vc.vehicleControlStatus != DispatcherAckMessageStatus {
		logger.Error("Need DispatcherAckMessageStatus!")
		return errors.New("Need DispatcherAckMessageStatus!")
	}

	vc.vehicleControlStatus = RemoteOperationEndStatus

	msg := message.Message{
		Connection: eve.Conn,
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
		MessageFlag:      VehicleControlMessageFlag,
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

	vc.timeoutTimer = time.NewTimer(VehicleControlTimeoutTime)

	if vc.closeTimeoutTimer == nil {
		vc.closeTimeoutTimer = make(chan bool, 1)
	}

	go func() {
		logger.Info("Start timer......")
		select {
		case <-vc.timeoutTimer.C:
			logger.Error("Timeout timer coming, vc fail!")
			//eve.PushEventChannel(EventVehicleControlRequest, nil)
			vc.vehicleControlStatus = VehicleControlStop
		case <-vc.closeTimeoutTimer:
			logger.Info("Close Timeout timer!")
		}

		logger.Info("Timer Close......")
	}()

	return nil
}

func (vc *VehicleControl) DispatcherAckMessage2(eve *Event, ackMsg *message.Message) error {
	if vc.vehicleControlStatus == RemoteOperationEndStatus {
		vc.timeoutTimer.Stop()

		//vc.vehicleControlStatus = DispatcherAckMessageStatus

		dispatcherAckMessageServData := &DispatcherAckMessageServData{}
		err := json.Unmarshal(ackMsg.ServData, dispatcherAckMessageServData)
		if err != nil {
			logger.Error(err)
			return err
		}

		logger.Debug("dispatcherAckMessageServData =", string(ackMsg.ServData))

		if dispatcherAckMessageServData.Operation != vc.operation {
			vc.closeTimeoutTimer <- true
			vc.vehicleControlStatus = VehicleControlStop

			return errors.New("operation not right!")
		}

		eve.PushEventChannel(EventRemoteOperationAck, ackMsg)
	} else if vc.vehicleControlStatus == RemoteOperationAckStatus {
		vc.timeoutTimer.Stop()

		dispatcherAckMessageServData := &DispatcherAckMessageServData{}
		err := json.Unmarshal(ackMsg.ServData, dispatcherAckMessageServData)
		if err != nil {
			logger.Error(err)
			return err
		}

		logger.Debug("dispatcherAckMessageServData =", string(ackMsg.ServData))

		vc.closeTimeoutTimer <- true
		vc.vehicleControlStatus = VehicleControlStop
	} else {
		logger.Error("Need RemoteOperationEndStatus!")
		return errors.New("Need RemoteOperationEndStatus!")
	}

	return nil
}

func (vc *VehicleControl) RemoteOperationAck(eve *Event, ackMsg *message.Message) error {
	if vc.vehicleControlStatus != RemoteOperationEndStatus {
		logger.Error("Need RemoteOperationEndStatus!")
		return errors.New("Need RemoteOperationEndStatus!")
	}

	vc.vehicleControlStatus = RemoteOperationAckStatus

	msg := message.Message{
		Connection: eve.Conn,
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
		MessageFlag:      VehicleControlMessageFlag,
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

	vc.timeoutTimer.Reset(VehicleControlTimeoutTime)

	return nil
}
