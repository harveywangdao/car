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
	VehicleInfoUploadTimeoutTime = 5 * time.Second
	VehicleInfoUploadMessageFlag = 0x1

	VHSUpdateMessageAid = 0xF5
	VHSUpdateMessageMid = 0x4

	VHSUpdateMessageAckAid = 0xF5
	VHSUpdateMessageAckMid = 0x2
)

const (
	VHSUpdateMessageStop = iota
	VHSUpdateMessageStatus
	/*VHSUpdateMessageAckStatus*/
)

type VehicleInfoUpload struct {
	vehicleInfoUploadTimer      *time.Timer
	closeVehicleInfoUploadTimer chan bool

	vehicleInfoUploadStatus int
}

type VHSUpdateMessageServData struct {
}

func (viu *VehicleInfoUpload) VHSUpdateMessage(eve *Event) error {
	if viu.vehicleInfoUploadStatus != VHSUpdateMessageStop {
		logger.Error("VehicleInfoUpload already start!")
		return errors.New("VehicleInfoUpload already start!")
	}

	viu.vehicleInfoUploadStatus = VHSUpdateMessageStatus

	msg := message.Message{
		Connection: eve.Conn,
	}

	//Service data
	vehicleInfoUploadReqServData := VHSUpdateMessageServData{}

	serviceData, err := json.Marshal(&vehicleInfoUploadReqServData)
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
		Aid:                  VHSUpdateMessageAid,
		Mid:                  VHSUpdateMessageMid,
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
		MessageFlag:      VehicleInfoUploadMessageFlag,
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

	logger.Debug("Send VHSUpdateMessage Success---")

	viu.vehicleInfoUploadTimer = time.NewTimer(VehicleInfoUploadTimeoutTime)

	if viu.closeVehicleInfoUploadTimer == nil {
		viu.closeVehicleInfoUploadTimer = make(chan bool, 1)
	}

	go func() {
		logger.Debug("Start timer......")
		select {
		case <-viu.vehicleInfoUploadTimer.C:
			logger.Info("Timer coming, need VehicleInfoUpload again!")

			viu.vehicleInfoUploadStatus = VHSUpdateMessageStop
			eve.PushEventChannel(EventVHSUpdateMessage, nil)

		case <-viu.closeVehicleInfoUploadTimer:
			logger.Info("Close VehicleInfoUpload timer!")
		}

		logger.Info("Timer Close......")
	}()

	return nil
}

func (viu *VehicleInfoUpload) VHSUpdateMessageAck(eve *Event, ackMsg *message.Message) error {
	if viu.vehicleInfoUploadStatus != VHSUpdateMessageStatus {
		logger.Error("Need VHSUpdateMessage!")
		return errors.New("Need VHSUpdateMessage!")
	}

	viu.vehicleInfoUploadTimer.Stop()
	viu.closeVehicleInfoUploadTimer <- true

	viu.vehicleInfoUploadStatus = VHSUpdateMessageStop

	return nil
}
