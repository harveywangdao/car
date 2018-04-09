package gateway

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

func (vc *VehicleControl) RemoteOperationReq(tbox *Tbox, op string) error {
	if vc.vehicleControlStatus != VehicleControlStop {
		logger.Error("VehicleControl already start!")
		return errors.New("VehicleControl already start!")
	}

	vc.vehicleControlStatus = RemoteOperationReqStatus

	msg := message.Message{
		Connection: tbox.Conn,
	}

	//Service data
	logger.Info("Operation =", op)
	operationValue, err := vc.convertOperation(op)
	if err != nil {
		logger.Error(err)
		return err
	}

	vc.operation = operationValue

	remoteOperationReqServData := RemoteOperationReqServData{
		Operation:          vc.operation,
		OperationParameter: 0, //need fix
	}

	serviceData, err := json.Marshal(&remoteOperationReqServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("serviceData =", serviceData)
	logger.Debug("serviceDataJson =", string(serviceData))

	aesKey, err := tbox.GetAesKey()
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
		Aid:                  RemoteOperationRequestAid,
		Mid:                  RemoteOperationRequestMid,
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
		Bid:              tbox.GetBid(),
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

	logger.Debug("Send RemoteOperationReq Success---")

	vc.timeoutTimer = time.NewTimer(VehicleControlTimeoutTime)

	if vc.closeTimeoutTimer == nil {
		vc.closeTimeoutTimer = make(chan bool, 1)
	}

	go func() {
		logger.Info("Start timer......")
		select {
		case <-vc.timeoutTimer.C:
			logger.Error("Timeout timer coming, vc fail!")
			//tbox.PushEventChannel(EventVehicleControlRequest, nil)
			vc.vehicleControlStatus = VehicleControlStop
		case <-vc.closeTimeoutTimer:
			logger.Info("Close Timeout timer!")
		}

		logger.Info("Timer Close......")
	}()

	return nil
}

func (vc *VehicleControl) DispatcherAckMessage(tbox *Tbox, ackMsg *message.Message) error {
	if vc.vehicleControlStatus != RemoteOperationReqStatus {
		logger.Error("Need RemoteOperationReqStatus!")
		return errors.New("Need RemoteOperationReqStatus!")
	}

	vc.timeoutTimer.Stop()

	vc.vehicleControlStatus = DispatcherAckMessageStatus

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

	return nil
}

func (vc *VehicleControl) RemoteOperationEnd(tbox *Tbox, endMsg *message.Message) error {
	if vc.vehicleControlStatus != DispatcherAckMessageStatus {
		logger.Error("Need DispatcherAckMessageStatus!")
		return errors.New("Need DispatcherAckMessageStatus!")
	}

	vc.vehicleControlStatus = RemoteOperationEndStatus

	remoteOperationEndServData := &RemoteOperationEndServData{}
	err := json.Unmarshal(endMsg.ServData, remoteOperationEndServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	if remoteOperationEndServData.Operation != vc.operation {
		vc.closeTimeoutTimer <- true
		vc.vehicleControlStatus = VehicleControlStop

		return errors.New("operation not right!")
	}

	//remoteOperationEndServData.Parameter

	err = vc.dispatcherAckMessage1(tbox, endMsg)

	return nil
}

func (vc *VehicleControl) dispatcherAckMessage1(tbox *Tbox, respMsg *message.Message) error {
	msg := message.Message{
		Connection: tbox.Conn,
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

	aesKey, err := tbox.GetAesKey()
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
		EventCreationTime:    respMsg.DisPatch.EventCreationTime,
		Aid:                  DispatcherAckMessageAid,
		Mid:                  DispatcherAckMessageMid,
		MessageCounter:       respMsg.DisPatch.MessageCounter + 1,
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
		Bid:              respMsg.MesHeader.Bid,
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

func (vc *VehicleControl) RemoteOperationAck(tbox *Tbox, ackMsg *message.Message) error {
	if vc.vehicleControlStatus != RemoteOperationEndStatus {
		logger.Error("Need RemoteOperationEndStatus!")
		return errors.New("Need RemoteOperationEndStatus!")
	}

	vc.vehicleControlStatus = RemoteOperationAckStatus

	remoteOperationAckServData := &RemoteOperationAckServData{}
	err := json.Unmarshal(ackMsg.ServData, remoteOperationAckServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	if remoteOperationAckServData.Operation != vc.operation {
		vc.closeTimeoutTimer <- true
		vc.vehicleControlStatus = VehicleControlStop

		return errors.New("operation not right!")
	}

	if remoteOperationAckServData.Status != RemoteOperationSuccess {
		vc.closeTimeoutTimer <- true
		vc.vehicleControlStatus = VehicleControlStop

		return errors.New("operation not right!")
	}

	err = vc.dispatcherAckMessage1(tbox, ackMsg)

	vc.closeTimeoutTimer <- true
	vc.vehicleControlStatus = VehicleControlStop

	return nil
}
