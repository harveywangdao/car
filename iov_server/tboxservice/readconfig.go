package tboxservice

import (
	"encoding/json"
	"errors"
	/*"gopkg.in/mgo.v2/bson"*/
	"github.com/jinzhu/gorm"
	"hcxy/iov/database"
	"hcxy/iov/database/mongo"
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

	readConfReqServData *ReadConfigReqServData
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

func (readConf *ReadConfig) GetIndexList() []byte {
	Indexs := make([]byte, 8)
	Indexs[0] = VehicleStatusInforIndex
	Indexs[1] = EngineStatusInforIndex
	Indexs[2] = GPSIndex
	return Indexs
}

func (readConf *ReadConfig) ReadConfigReq(eve *Event) error {
	if readConf.readConfigStatus != ReadConfigStop {
		logger.Error("ReadConfig already start!")
		return errors.New("ReadConfig already start!")
	}

	/*	if eve.TboxStatus != TboxRegisteredLogined{
		logger.Error("Not login or register")
		return errors.New("Not login or register")
	}*/

	readConf.readConfigStatus = ReadConfigReqStatus

	msg := message.Message{
		Connection: eve.Conn,
	}

	//Service data
	readConf.readConfReqServData = &ReadConfigReqServData{
		CaseID:    0,
		IndexList: readConf.GetIndexList(),
	}

	serviceData, err := json.Marshal(readConf.readConfReqServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("serviceData =", serviceData)
	logger.Info("serviceDataJson =", string(serviceData))

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
		Aid:                  ReadConfigReqAid,
		Mid:                  ReadConfigReqMid,
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

	logger.Debug("Send ReadConfigReq Success---")

	readConf.readConfTimeoutTimer = time.NewTimer(ReadConfigTimeoutTime)

	if readConf.closeTimeoutTimer == nil {
		readConf.closeTimeoutTimer = make(chan bool, 1)
	}

	go func() {
		logger.Info("Start timer......")
		select {
		case <-readConf.readConfTimeoutTimer.C:
			logger.Info("Timeout timer coming, readConf fail!")
			eve.PushEventChannel(EventReadConfigRequest, nil)
			readConf.readConfigStatus = ReadConfigStop
		case <-readConf.closeTimeoutTimer:
			logger.Info("Close Timeout timer!")
		}

		logger.Info("Timer Close......")
	}()

	return nil
}

func (readConf *ReadConfig) saveVehicleStatusInfor(bid uint32, infor *vehicle.VehicleStatusInfor) error {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	var id int
	err = tboxDB.QueryRow("SELECT id FROM tboxbaseinfo_tbl WHERE bid = ?", bid).Scan(
		&id)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("id =", id)

	stmtIns, err := tboxDB.Prepare(`INSERT INTO VehicleStatusInfor_tbl(
		ResidualMileage,
		OdographMeter,
		MaintainMileage,
		Fuel,
		AverageFuleCut,
		AverageSpeed,
		VehicleInstantaneousSpeed,
		WheelSpeed_FL,
		WheelSpeed_FR,
		WheelSpeed_RL,
		WheelSpeed_RR,
		WheelTyrePressure_FL,
		WheelTyrePressure_FR,
		WheelTyrePressure_RL,
		WheelTyrePressure_RR,
		WheelTyreTemperature_FL,
		WheelTyreTemperature_FR,
		WheelTyreTemperature_RL,
		WheelTyreTemperature_RR,
		SteeringAngularSpeed,
		SteeringAngle,
		tbox_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer stmtIns.Close()

	_, err = stmtIns.Exec(
		infor.ResidualMileage,
		infor.OdographMeter,
		infor.MaintainMileage,
		infor.Fuel,
		infor.AverageFuleCut,
		infor.AverageSpeed,
		infor.VehicleInstantaneousSpeed,
		infor.WheelSpeed.WheelSpeed_FL,
		infor.WheelSpeed.WheelSpeed_FR,
		infor.WheelSpeed.WheelSpeed_RL,
		infor.WheelSpeed.WheelSpeed_RR,
		infor.TyrePressure.WheelTyrePressure_FL,
		infor.TyrePressure.WheelTyrePressure_FR,
		infor.TyrePressure.WheelTyrePressure_RL,
		infor.TyrePressure.WheelTyrePressure_RR,
		infor.TyreTemperature.WheelTyreTemperature_FL,
		infor.TyreTemperature.WheelTyreTemperature_FR,
		infor.TyreTemperature.WheelTyreTemperature_RL,
		infor.TyreTemperature.WheelTyreTemperature_RR,
		infor.SteeringAngularSpeed,
		infor.SteeringAngle,
		id)
	if err != nil {
		logger.Error(err)
		return err
	}

	session, err := mongo.CloneMgoSession()
	if err != nil {
		logger.Error(err)
		return err
	}
	defer session.Close()

	data := &struct {
		*vehicle.VehicleStatusInfor
		Tbox_id int
	}{
		infor,
		id,
	}

	c := session.DB("iovdb").C("VehicleStatusInfor")
	err = c.Insert(data)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("Save to mongo!")

	return nil
}

func (readConf *ReadConfig) saveEngineStatusInfor(bid uint32, infor *vehicle.EngineStatusInfor) error {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	var id int
	err = tboxDB.QueryRow("SELECT id FROM tboxbaseinfo_tbl WHERE bid = ?", bid).Scan(
		&id)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("id =", id)

	stmtIns, err := tboxDB.Prepare(`INSERT INTO EngineStatusInfor_tbl(
		EngineSpeed,
		EngineRunTime,
		VehicleRunTime,
		EngState,
		GearLevPos,
		tbox_id) VALUES(?,?,?,?,?,?)`)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer stmtIns.Close()

	_, err = stmtIns.Exec(
		infor.EngineSpeed,
		infor.EngineRunTime,
		infor.VehicleRunTime,
		infor.EngState,
		infor.GearLevPos,
		id)
	if err != nil {
		logger.Error(err)
		return err
	}

	session, err := mongo.CloneMgoSession()
	if err != nil {
		logger.Error(err)
		return err
	}
	defer session.Close()

	data := &struct {
		*vehicle.EngineStatusInfor
		Tbox_id int
	}{
		infor,
		id,
	}

	c := session.DB("iovdb").C("EngineStatusInfor")
	err = c.Insert(data)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("Save to mongo!")

	return nil
}

func (readConf *ReadConfig) saveGpsInfor(bid uint32, infor *vehicle.GPS) error {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	var id int
	err = tboxDB.QueryRow("SELECT id FROM tboxbaseinfo_tbl WHERE bid = ?", bid).Scan(
		&id)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("id =", id)

	stmtIns, err := tboxDB.Prepare(`INSERT INTO VehicleGpsInfor_tbl(
		Latitude,
		Longitude,
		Altitude,
		Speed,
		Bearing,
		Accuracy,
		Time,
		tbox_id) VALUES(?,?,?,?,?,?,?,?)`)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer stmtIns.Close()

	_, err = stmtIns.Exec(
		infor.Latitude,
		infor.Longitude,
		infor.Altitude,
		infor.Speed,
		infor.Bearing,
		infor.Accuracy,
		infor.Time,
		id)
	if err != nil {
		logger.Error(err)
		return err
	}

	session, err := mongo.CloneMgoSession()
	if err != nil {
		logger.Error(err)
		return err
	}
	defer session.Close()

	data := &struct {
		*vehicle.GPS
		Tbox_id int
	}{
		infor,
		id,
	}

	c := session.DB("iovdb").C("gpsinfor")
	err = c.Insert(data)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("Save to mongo!")

	type GpsInfor struct {
		Latitude  float64
		Longitude float64
		Altitude  float64
		Speed     float64
		Bearing   float64
		Accuracy  float64
		Time      int64
		Tbox_id   int
	}

	ormData := GpsInfor{
		infor.Latitude,
		infor.Longitude,
		infor.Altitude,
		infor.Speed,
		infor.Bearing,
		infor.Accuracy,
		infor.Time,
		id,
	}

	db, err := gorm.Open("mysql", "root:123456@/iovdb?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		logger.Error(err)
		return err
	}
	defer db.Close()

	//db.DropTable(&ormData)

	if !db.HasTable(&ormData) {
		db.CreateTable(&ormData)
	}

	db.Create(&ormData)

	logger.Info("Save to ORM mysql!")

	return nil
}

func (readConf *ReadConfig) saveVechileInfor(bid uint32, vInfor map[string][]byte) error {
	for k, v := range vInfor {
		switch k {
		case VehicleStatusInfor:
			vehicleStatusInfor := vehicle.VehicleStatusInfor{}
			err := json.Unmarshal(v, &vehicleStatusInfor)
			if err != nil {
				logger.Error(err)
				return err
			}

			logger.Info("vehicleStatusInfor =", vehicleStatusInfor)
			err = readConf.saveVehicleStatusInfor(bid, &vehicleStatusInfor)
			if err != nil {
				logger.Error(err)
				return err
			}

		case EngineStatusInfor:
			engineStatusInfor := vehicle.EngineStatusInfor{}
			err := json.Unmarshal(v, &engineStatusInfor)
			if err != nil {
				logger.Error(err)
				return err
			}

			logger.Info("engineStatusInfor =", engineStatusInfor)
			err = readConf.saveEngineStatusInfor(bid, &engineStatusInfor)
			if err != nil {
				logger.Error(err)
				return err
			}

		case GPS:
			gps := vehicle.GPS{}
			err := json.Unmarshal(v, &gps)
			if err != nil {
				logger.Error(err)
				return err
			}

			logger.Info("gps =", gps)
			err = readConf.saveGpsInfor(bid, &gps)
			if err != nil {
				logger.Error(err)
				return err
			}

		default:
			logger.Error("Not support item!")
			continue
		}
	}

	return nil
}

func (readConf *ReadConfig) ReadConfigAck(eve *Event, ackMsg *message.Message) error {
	if readConf.readConfigStatus != ReadConfigReqStatus {
		logger.Error("Need ReadConfigReq!")
		return errors.New("Need ReadConfigReq!")
	}

	readConf.readConfTimeoutTimer.Stop()
	readConf.closeTimeoutTimer <- true

	readConfAckServData := &ReadConfigAckServData{}
	err := json.Unmarshal(ackMsg.ServData, readConfAckServData)
	if err != nil {
		logger.Error(err)
		readConf.readConfigStatus = ReadConfigStop
		return err
	}

	logger.Info("readConfAckServData =", string(ackMsg.ServData))
	logger.Info("WorkConfigList =", readConfAckServData.WorkConfigList)

	err = readConf.saveVechileInfor(ackMsg.MesHeader.Bid, readConfAckServData.WorkConfigList)
	if err != nil {
		logger.Error(err)
		readConf.readConfigStatus = ReadConfigStop
		return err
	}

	readConf.readConfigStatus = ReadConfigStop

	return nil
}
