package message

import (
	"encoding/binary"
	"errors"
	"hcxy/iov/crypto/aes"
	"hcxy/iov/log/logger"
	"hcxy/iov/util"
	"time"
)

const (
	Encrypt_No     uint8 = 0x0
	Encrypt_AES128 uint8 = 0x1
	Encrypt_Base64 uint8 = 0x3

	MessageHeaderID    uint32 = 0x74426F78
	MessageHeaderIDLen int    = 4
	AES128KEY                 = "1234567890123456"

	RetSuccess               = 0
	ErrorCodeGeneral         = 1
	ErrorCodeConnectionBreak = 2
)

type MessageHeader struct {
	FixHeader        uint32 //0x74,0x42,0x6F,0x78
	ServiceDataCheck uint8  //ServiceData ^
	ServiceVersion   uint8
	Bid              uint32
	MessageFlag      uint8
}

type DispatchData struct {
	EventCreationTime    uint32
	Aid                  uint8
	Mid                  uint8
	MessageCounter       uint16
	ServiceDataLength    uint16
	Result               uint8
	SecurityVersion      uint8 //0x0:no encrypt; 0x1:AES128; 0x3:Base64
	DispatchCreationTime uint32
}

type MessageConn interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
}

type Message struct {
	MesHeader  MessageHeader
	DisPatch   DispatchData
	ServData   []byte
	CheckSum   byte
	Connection MessageConn

	Aes128Key  string
	CallbackFn func(uint32) string
}

func (msg *Message) getAES128Key(bid uint32) string {
	//return AES128KEY
	return msg.Aes128Key
}

/////////////////////////////////////Receive Message//////////////////////////////////////////////////
func (msg *Message) checkMessageChecksum(message []byte) bool {
	checkSum := util.DataXOR(message[:len(message)-1])
	return (checkSum == message[len(message)-1])
}

func (msg *Message) RecvOneMessage() ([]byte, int, error) {
	//Read message header id 0x574D5442
	var n int
	var err error
	buf := make([]byte, MessageHeaderIDLen)
	readData := make([]byte, 0, 4096)

	for {
		n, err = msg.Connection.Read(buf)
		if err != nil {
			logger.Error(err)
			return nil, ErrorCodeConnectionBreak, err
		}

		logger.Debug("Read data length =", n)

		readData = append(readData, buf[:n]...)

		if len(readData) >= MessageHeaderIDLen {
			var messageHeaderId uint32
			if err = util.ByteSliceToStruct(readData[:MessageHeaderIDLen], &messageHeaderId); err != nil {
				logger.Error(err)
				return nil, ErrorCodeGeneral, err
			}

			logger.Debug("messageHeaderId =", messageHeaderId)

			if MessageHeaderID == messageHeaderId {
				break
			} else {
				return nil, ErrorCodeGeneral, errors.New("Not true message!")
			}
		}
	}

	//Read message header and dispatch data
	var mh MessageHeader
	var dd DispatchData
	var messageHeaderLen int = binary.Size(mh)
	var dispatchLen int = binary.Size(dd)

	buf = make([]byte, messageHeaderLen+dispatchLen-MessageHeaderIDLen)
	for {
		n, err = msg.Connection.Read(buf)
		if err != nil {
			logger.Error(err)
			return nil, ErrorCodeConnectionBreak, err
		}

		logger.Debug("Read data length =", n)

		readData = append(readData, buf[:n]...)

		if len(readData) >= messageHeaderLen+dispatchLen {
			if err = util.ByteSliceToStruct(readData[messageHeaderLen:messageHeaderLen+dispatchLen], &dd); err != nil {
				logger.Error(err)
				return nil, ErrorCodeGeneral, err
			}

			logger.Debug("dd =", dd)
			break
		}
	}

	//Get all message data
	oneMsgLen := messageHeaderLen + dispatchLen + int(dd.ServiceDataLength) + 1
	buf = make([]byte, int(dd.ServiceDataLength)+1)
	for {
		n, err = msg.Connection.Read(buf)
		if err != nil {
			logger.Error(err)
			return nil, ErrorCodeConnectionBreak, err
		}

		logger.Debug("Read data length =", n)

		readData = append(readData, buf[:n]...)

		if len(readData) >= oneMsgLen {
			break
		}
	}

	//Check data validity
	if !msg.checkMessageChecksum(readData[:oneMsgLen]) {
		return nil, ErrorCodeGeneral, errors.New("Not true message!")
	}

	return readData[:oneMsgLen], RetSuccess, nil
}

func (msg *Message) ParseOneMessage(originMessageData []byte) ([]byte, error) {
	var messageHeaderLen int = binary.Size(msg.MesHeader)
	var dispatchDataLen int = binary.Size(msg.DisPatch)
	var err error

	if !msg.checkMessageChecksum(originMessageData) {
		return nil, errors.New("Not true message!")
	}

	msg.CheckSum = originMessageData[len(originMessageData)-1]

	//Parse message header
	if err = util.ByteSliceToStruct(originMessageData[:messageHeaderLen], &msg.MesHeader); err != nil {
		logger.Error(err)
		return nil, err
	}

	logger.Debug("msg.MesHeader =", msg.MesHeader)

	if msg.MesHeader.FixHeader != MessageHeaderID {
		return nil, errors.New("Not true message!")
	}

	//Parse dispatch data
	dispatchData := originMessageData[messageHeaderLen : messageHeaderLen+dispatchDataLen]
	if err = util.ByteSliceToStruct(dispatchData, &msg.DisPatch); err != nil {
		logger.Error(err)
		return nil, err
	}

	logger.Debug("msg.DisPatch =", msg.DisPatch)

	//Decrypt service data
	switch msg.DisPatch.SecurityVersion {
	case Encrypt_No:
		msg.ServData = originMessageData[messageHeaderLen+dispatchDataLen : len(originMessageData)-1]
	case Encrypt_AES128:
		msg.ServData, err = aes.AesDecrypt(originMessageData[messageHeaderLen+dispatchDataLen:len(originMessageData)-1],
			[]byte(msg.CallbackFn(msg.MesHeader.Bid)))
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	case Encrypt_Base64:
		fallthrough
	default:
		return nil, errors.New("Not support this encrypt!")
	}

	//Check ServiceDataCheck
	serviceDataCheck := util.DataXOR(msg.ServData)
	if serviceDataCheck != msg.MesHeader.ServiceDataCheck {
		logger.Error("Service data check error!")
		//return nil, errors.New("service data check error!")
		return msg.ServData, nil
	}

	return msg.ServData, nil
}

func (msg *Message) RecvMessage() (int, error) {
	originMessageData, errorCode, err := msg.RecvOneMessage()
	if err != nil {
		logger.Error(err)
		return errorCode, err
	}

	logger.Debug("originMessageData =", originMessageData)

	serviceData, err := msg.ParseOneMessage(originMessageData)
	if err != nil {
		logger.Error(err)
		return ErrorCodeGeneral, err
	}

	logger.Debug("serviceData =", serviceData)

	return RetSuccess, nil
}

/////////////////////////////////////Send Message//////////////////////////////////////////////////
/*type FnGetServiceData func() ([]byte, error)
type FnGetDisatchData func() ([]byte, error)
type FnGetMessageHeaderData func(byte, int, int) ([]byte, error)
*/
func (msg *Message) GetServiceData() ([]byte, error) {
	var serviceData []byte
	serviceData = make([]byte, 256)
	for i := 0; i < len(serviceData); i++ {
		serviceData[i] = byte(i)
	}

	logger.Debug("serviceData =", serviceData)

	return serviceData, nil
}

func (msg *Message) GetDispatchData(encryptServData []byte) ([]byte, error) {
	var dd DispatchData
	dd.EventCreationTime = uint32(time.Now().Unix()) //not sure
	dd.Aid = 0x5
	dd.Mid = 0x2
	dd.MessageCounter = 0 //not sure
	dd.ServiceDataLength = uint16(len(encryptServData))
	dd.Result = 0
	dd.SecurityVersion = Encrypt_AES128 //AES-128
	dd.DispatchCreationTime = uint32(time.Now().Unix())

	dispatchData, err := util.StructToByteSlice(dd)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return dispatchData, nil
}

func (msg *Message) GetMessageHeaderData(serviceData []byte) ([]byte, error) {
	var mh MessageHeader
	mh.FixHeader = MessageHeaderID
	mh.ServiceDataCheck = util.DataXOR(serviceData)
	mh.ServiceVersion = 0x0 //not sure
	mh.Bid = 0x0            //not sure
	mh.MessageFlag = 0x0

	messageHeaderData, err := util.StructToByteSlice(mh)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return messageHeaderData, nil
}

func (msg *Message) EncryptServiceData(encryptType uint8, key string, serviceData []byte) ([]byte, error) {
	var encryptServiceData []byte
	var err error

	switch encryptType {
	case Encrypt_AES128:
		encryptServiceData, err = aes.AesEncrypt(serviceData, []byte(key)) //msg.getAES128Key(0)
		if err != nil {
			logger.Error(err)
			return nil, err
		}

	case Encrypt_No:
		encryptServiceData = serviceData
	case Encrypt_Base64:
		fallthrough
	default:
		return serviceData, errors.New("Encrypt Type Unknown!")
	}

	return encryptServiceData, nil
}

func (msg *Message) GetOneMessage(msgHeaderData, DispatchData, encryptServiceData []byte) ([]byte, error) {
	headerDispatchService := append(msgHeaderData, DispatchData...)
	headerDispatchService = append(headerDispatchService, encryptServiceData...)

	var checkSum byte = util.DataXOR(headerDispatchService)
	logger.Debug("checkSum =", checkSum)

	data := append(headerDispatchService, checkSum)
	logger.Debug("Data len =", len(data))
	logger.Debug("Data =", data)

	return data, nil
}

func (msg *Message) SendOneMessage(data []byte) error {
	n, err := msg.Connection.Write(data)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	logger.Debug("send data length =", n)
	return nil
}

func (msg *Message) SendMessage(msgHeaderData, dispatchData, encryptServData []byte) error {
	data, err := msg.GetOneMessage(msgHeaderData, dispatchData, encryptServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("data =", data)
	err = msg.SendOneMessage(data)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}
