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

func (viu *VehicleInfoUpload) VHSUpdateMessage(tbox *Tbox, reqMsg *message.Message) error {
	if viu.vehicleInfoUploadStatus != VHSUpdateMessageStop {
		logger.Error("VehicleInfoUpload already start!")
		return errors.New("VehicleInfoUpload already start!")
	}

	viu.vehicleInfoUploadStatus = VHSUpdateMessageStatus

	vhsUpdateMessageServData := &VHSUpdateMessageServData{}
	err := json.Unmarshal(reqMsg.ServData, vhsUpdateMessageServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	//vhsUpdateMessageServData

	logger.Debug("vhsUpdateMessageServData =", string(reqMsg.ServData))

	tbox.PushEventChannel(EventVHSUpdateMessageAck, reqMsg)

	return nil
}

func (viu *VehicleInfoUpload) VHSUpdateMessageAck(tbox *Tbox, reqMsg *message.Message) error {
	if viu.vehicleInfoUploadStatus != VHSUpdateMessageStatus {
		logger.Error("Need VHSUpdateMessage!")
		return errors.New("Need VHSUpdateMessage!")
	}

	msg := message.Message{
		Connection: tbox.Conn,
	}

	//Service data
	serviceData := make([]byte, 1)
	logger.Debug("serviceData =", serviceData)

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
		EventCreationTime:    reqMsg.DisPatch.EventCreationTime,
		Aid:                  VHSUpdateMessageAckAid,
		Mid:                  VHSUpdateMessageAckMid,
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

	logger.Debug("Send VHSUpdateMessageAck Success---")

	viu.vehicleInfoUploadStatus = VHSUpdateMessageStop

	return nil
}
