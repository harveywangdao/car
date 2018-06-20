package gateway

import (
	"encoding/hex"
	"encoding/json"
	"github.com/harveywangdao/road/crypto/md5"
	"github.com/harveywangdao/road/database"
	"github.com/harveywangdao/road/log/logger"
	"github.com/harveywangdao/road/message"
	"github.com/harveywangdao/road/util"
	"time"
)

import (
	pb "github.com/harveywangdao/road/protobuf/adderserver"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	RegisterSuccess byte = 0x78
	RegisterFailure byte = 0xA8
	AlreadyRegister byte = 0x79
)

type Register struct {
	registerReqData *RegisterReqData
}

type RegisterReqData struct {
	PerAesKey  string `json:"peraeskey"`
	ThingId    string `json:"thingid"`
	TBoxSN     string `json:"tboxsn"`
	IMSI       string `json:"imsi"`
	RollNumber string `json:"rollnumber"`
	ICCID      string `json:"iccid"`
}

type RegisterAckMsg struct {
	Status      byte   `json:"status"`
	CallbackNum string `json:"callbacknumber"`
	Bid         uint32 `json:"bid"`
}

func (re *Register) genCallbackNum() string {
	data := md5.GenMd5([]byte(re.registerReqData.PerAesKey + re.registerReqData.RollNumber))
	cbn := util.Substr(hex.EncodeToString(data), 0, 16)

	logger.Debug("data =", data, "hex string(data) =", hex.EncodeToString(data))
	logger.Debug("cbn =", cbn)

	return cbn
}

func (re *Register) checkRegisterData() byte {
	result := RegisterFailure //fail

	db, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return result
	}

	var thingserialno, prethingaes128key, thingid, iccid, imsi, thingaes128key string
	var id, status, bid, eventCreationTime int
	err = db.QueryRow("SELECT * FROM thingbaseinfodata_tbl WHERE thingid = ?", re.registerReqData.ThingId).Scan(
		&id,
		&thingserialno,
		&prethingaes128key,
		&thingid,
		&iccid,
		&imsi,
		&status,
		&bid,
		&thingaes128key,
		&eventCreationTime)

	if err != nil {
		logger.Error(err)
		return result
	}

	if re.registerReqData.PerAesKey != prethingaes128key {
		return result
	}

	if re.registerReqData.TBoxSN != thingserialno {
		return result
	}

	if re.registerReqData.IMSI != imsi && re.registerReqData.ICCID != iccid {
		return result
	}

	if status == ThingRegisteredUnLogin || status == ThingRegisteredLogined {
		result = AlreadyRegister //already register
		return result
	} else {
		result = RegisterSuccess //success
	}

	return result
}

func (re *Register) registerThing(bid, eventCreationTime uint32, newAesKey string) error {
	db, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	stmtUpd, err := db.Prepare("UPDATE thingbaseinfodata_tbl SET status=?,bid=?,thingaes128key=?,eventcreationtime=? where thingid=?")
	if err != nil {
		logger.Error(err)
		return err
	}
	defer stmtUpd.Close()

	_, err = stmtUpd.Exec(ThingRegisteredUnLogin, int(bid), newAesKey, eventCreationTime, re.registerReqData.ThingId)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func (re *Register) getDispatchData(regReqMsg *message.Message, encryptServData []byte, result byte) ([]byte, error) {
	var dd message.DispatchData
	dd.EventCreationTime = regReqMsg.DisPatch.EventCreationTime
	dd.Aid = 0x1
	dd.Mid = 0x2
	dd.MessageCounter = regReqMsg.DisPatch.MessageCounter + 1
	dd.ServiceDataLength = uint16(len(encryptServData))
	dd.Result = result
	dd.SecurityVersion = message.Encrypt_No
	dd.DispatchCreationTime = uint32(time.Now().Unix())

	dispatchData, err := util.StructToByteSlice(dd)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return dispatchData, nil
}

func (re *Register) getMessageHeaderData(serviceData []byte, bid uint32) ([]byte, error) {
	var mh message.MessageHeader
	mh.FixHeader = message.MessageHeaderID
	mh.ServiceDataCheck = util.DataXOR(serviceData)
	mh.ServiceVersion = 0x0 //not sure
	mh.Bid = bid
	mh.MessageFlag = 0x0

	messageHeaderData, err := util.StructToByteSlice(mh)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return messageHeaderData, nil
}

func (re *Register) genBid() uint32 {
	db, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return 0
	}

	var id int
	err = db.QueryRow("SELECT id FROM thingbaseinfodata_tbl WHERE thingid = ?", re.registerReqData.ThingId).Scan(&id)
	if err != nil {
		logger.Error(err)
		return 0
	}

	return uint32(id)
}

func (re *Register) RegisterACK(conn message.MessageConn, regReqMsg *message.Message) error {
	msg := message.Message{
		Connection: conn,
	}

	var result byte
	var bid uint32 = 0
	var callbackNum string = ""

	//Check data validity
	re.registerReqData = &RegisterReqData{}
	err := json.Unmarshal(regReqMsg.ServData, re.registerReqData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("re.registerReqData =", string(regReqMsg.ServData))

	result = re.checkRegisterData()
	if result == RegisterSuccess || result == AlreadyRegister {
		callbackNum = re.genCallbackNum()
		bid = re.genBid()

		err := re.registerThing(bid, regReqMsg.DisPatch.EventCreationTime, callbackNum)
		if err != nil {
			logger.Error(err)
			return err
		}
	}

	//Service data
	registerAckMsg := &RegisterAckMsg{}

	if result == RegisterSuccess {
		registerAckMsg.Status = 0
	} else {
		registerAckMsg.Status = 1
	}

	registerAckMsg.CallbackNum = callbackNum
	registerAckMsg.Bid = bid

	serviceData, err := json.Marshal(registerAckMsg)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("serviceData =", serviceData)
	logger.Debug("serviceData =", string(serviceData))

	//Encrypy serviceData
	encryptServData := serviceData

	dispatchData, err := re.getDispatchData(regReqMsg, encryptServData, result)
	if err != nil {
		logger.Error(err)
		return err
	}

	messageHeaderData, err := re.getMessageHeaderData(serviceData, bid)
	if err != nil {
		logger.Error(err)
		return err
	}

	err = msg.SendMessage(messageHeaderData, dispatchData, encryptServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info(re.registerReqData.ThingId, "Register success!")

	err = re.testgrpc()
	if err != nil {
		logger.Error("GRPC ERROR!")
		return err
	}

	return nil
}

func (re *Register) testgrpc() error {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		logger.Error("did not connect:", err)
		return err
	}
	defer conn.Close()

	c := pb.NewAdderClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.Add(ctx, &pb.AddRequest{Addparameter1: 12, Addparameter2: 13})
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("GRPG RETURN:", r.Sum)

	return nil
}
