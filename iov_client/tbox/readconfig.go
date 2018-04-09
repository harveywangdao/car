package tbox

import (
	"encoding/json"
	"errors"
	"hcxy/iov/log/logger"
	"hcxy/iov/message"
	"hcxy/iov/util"
	"hcxy/iov/vehicle"
	"time"
)

const (
	ReadConfigTimeoutTime = 5 * time.Second

	ReadConfigMessageFlag = 0x0

	ReadConfigReqAid = 0x5
	ReadConfigReqMid = 0x1

	ReadConfigAckAid = 0x5
	ReadConfigAckMid = 0x2
)

const (
	ReadConfigStop = iota
	ReadConfigReqStatus
	/*ReLoginAckStatus*/
)

type ReadConfig struct {
	readConfTimeoutTimer *time.Timer
	closeTimeoutTimer    chan bool

	readConfigStatus int
}

type ReadConfigReqServData struct {
	CaseID    uint16 `json:"caseid"`
	IndexList []byte `json:"indexlist"`
}

type ReadConfigAckServData struct {
	WorkConfigList map[string][]byte `json:"workconfiglist"`
}

const (
	VehicleStatusInforIndex = iota + 1
	EngineStatusInforIndex
	GPSIndex
)

const (
	VehicleStatusInfor = "VehicleStatusInfor"
	EngineStatusInfor  = "EngineStatusInfor"
	GPS                = "GPS"
)

func (readConf *ReadConfig) GetVehicleStatusInfor() ([]byte, error) {
	fourWheelSpeeds := vehicle.FourWheelSpeeds{
		WheelSpeed_FL: 44.3655,
		WheelSpeed_FR: 4545.12,
		WheelSpeed_RL: 4564.25,
		WheelSpeed_RR: 895.54,
	}

	fourWheelTyrePressure := vehicle.FourWheelTyrePressure{
		WheelTyrePressure_FL: 12.3,
		WheelTyrePressure_FR: 12.3,
		WheelTyrePressure_RL: 12.3,
		WheelTyrePressure_RR: 12.3,
	}

	fourWheelTyreTemperature := vehicle.FourWheelTyreTemperature{
		WheelTyreTemperature_FL: 42.2243,
		WheelTyreTemperature_FR: 42.2,
		WheelTyreTemperature_RL: -42.2,
		WheelTyreTemperature_RR: 42.2,
	}

	vehicleInfor := vehicle.VehicleStatusInfor{
		ResidualMileage:           123,
		OdographMeter:             456,
		MaintainMileage:           678,
		Fuel:                      90,
		AverageFuleCut:            21.354,
		AverageSpeed:              123,
		VehicleInstantaneousSpeed: 151.54415,
		WheelSpeed:                fourWheelSpeeds,
		TyrePressure:              fourWheelTyrePressure,
		TyreTemperature:           fourWheelTyreTemperature,
		SteeringAngularSpeed:      4151,
		SteeringAngle:             -5241,
	}

	vehicleInforData, err := json.Marshal(&vehicleInfor)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	logger.Debug("serviceData =", vehicleInforData)
	logger.Debug("serviceDataJson =", string(vehicleInforData))

	return vehicleInforData, nil
}

func (readConf *ReadConfig) GetEngineStatusInfor() ([]byte, error) {
	engineInfor := vehicle.EngineStatusInfor{
		EngineSpeed:    44564.365,
		EngineRunTime:  546156,
		VehicleRunTime: 34235245,
		EngState:       vehicle.EngineOn,
		GearLevPos:     vehicle.GearLeverPos_R,
	}

	engineInforData, err := json.Marshal(&engineInfor)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	logger.Debug("serviceData =", engineInforData)
	logger.Debug("serviceDataJson =", string(engineInforData))

	return engineInforData, nil
}

func (readConf *ReadConfig) GetGPSInfor() ([]byte, error) {
	gpsInfor := vehicle.GPS{
		Latitude:  14.334534,
		Longitude: 554.334534,
		Altitude:  41564.34534,
		Speed:     114.343543,
		Bearing:   2123.3434534,
		Accuracy:  3.25654,
		Time:      int64(time.Now().Unix()),
	}

	gpsInforData, err := json.Marshal(&gpsInfor)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	logger.Debug("serviceData =", gpsInforData)
	logger.Debug("serviceDataJson =", string(gpsInforData))

	return gpsInforData, nil
}

func (readConf *ReadConfig) GetConfig(indexs []byte) map[string][]byte {
	dataMap := make(map[string][]byte)

	for _, ele := range indexs {
		switch ele {
		case VehicleStatusInforIndex:
			vehicleInforData, err := readConf.GetVehicleStatusInfor()
			if err != nil {
				logger.Error(err)
				return nil
			}
			dataMap[VehicleStatusInfor] = vehicleInforData

		case EngineStatusInforIndex:
			engineInforData, err := readConf.GetEngineStatusInfor()
			if err != nil {
				logger.Error(err)
				return nil
			}
			dataMap[EngineStatusInfor] = engineInforData

		case GPSIndex:
			gpsInforData, err := readConf.GetGPSInfor()
			if err != nil {
				logger.Error(err)
				return nil
			}
			dataMap[GPS] = gpsInforData

		default:
			logger.Error("Unknown config! index :", ele)
		}
	}

	return dataMap
}

func (readConf *ReadConfig) ReadConfigReq(eve *Event, reqMsg *message.Message) error {
	if readConf.readConfigStatus != ReadConfigStop {
		logger.Error("ReadConfig already start!")
		return errors.New("ReadConfig already start!")
	}

	if eve.TboxStatus != TboxRegisteredLogined {
		logger.Error("Not login or register")
		return errors.New("Not login or register")
	}

	readConf.readConfigStatus = ReadConfigReqStatus

	eve.PushEventChannel(EventReadConfigAck, reqMsg)

	return nil
}

func (readConf *ReadConfig) ReadConfigAck(eve *Event, reqMsg *message.Message) error {
	if readConf.readConfigStatus != ReadConfigReqStatus {
		logger.Error("Need ReadConfigReq!")
		return errors.New("Need ReadConfigReq!")
	}

	msg := message.Message{
		Connection: eve.Conn,
	}

	//Service data
	readConfReqServData := &ReadConfigReqServData{}
	err := json.Unmarshal(reqMsg.ServData, readConfReqServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("readConfReqServData =", string(reqMsg.ServData))

	readConfAckServData := ReadConfigAckServData{
		WorkConfigList: readConf.GetConfig(readConfReqServData.IndexList),
	}

	serviceData, err := json.Marshal(&readConfAckServData)
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
		Aid:                  ReadConfigAckAid,
		Mid:                  ReadConfigAckMid,
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
		MessageFlag:      ReadConfigMessageFlag,
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

	logger.Debug("Send ReadConfigAck Success---")

	readConf.readConfigStatus = ReadConfigStop

	return nil
}
