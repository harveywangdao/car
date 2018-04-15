package gateway

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/harveywangdao/road/crypto/md5"
	"github.com/harveywangdao/road/database"
	"github.com/harveywangdao/road/log/logger"
	"github.com/harveywangdao/road/message"
	"github.com/harveywangdao/road/util"
	"time"
)

const (
	TimeoutTime = 2 * time.Second

	LoginResultCodeSuccess       = 0x00
	LoginResultCodeSnVinErr      = 0xA8
	LoginResultCodeBidErrOrUnreg = 0xA9
	LoginResultCodeInterrupt     = 0xAA
	LoginResultCodeAbstractErr   = 0xAB
	LoginResultCodeAesOutOfDate  = 0xAC

	KeyTypePreAesKey     = 0
	KeyTypeCurrentAesKey = 1

	FailureCodeKeyErr       = 0
	FailureCodeKeyOutOfDate = 1
	FailureCodeSystemErr    = 2
)

const (
	LoginStop = iota
	LoginRequestStatus
	LoginChallengeStatus
	LoginResponseStatus
)

type Login struct {
	timeoutTimer      *time.Timer
	closeTimeoutTimer chan bool

	loginReqServData   *LoginReqServData
	loginChallServData *LoginChallengeServData
	loginRespServData  *LoginResponseServData

	loginFailureResult byte

	loginStatus int
}

type LoginReqServData struct {
	KeyType     byte   `json:"keytype"`
	ThingSN     string `json:"thingsn"`
	ThingId     string `json:"thingid"`
	ThingRandom string `json:"thingrandom"`
}

type LoginChallengeServData struct {
	ThingRandomMd5 string `json:"thingrandommd5"`
	PlatRandom     string `json:"platrandom"`
}

type LoginResponseServData struct {
	SerialUP  string `json:"serialup"`
	AccessKey string `json:"accesskey"`
}

type LoginFailureServData struct {
	FailureCode byte `json:"failurecode"`
}

type LoginSuccessServData struct {
	AesRandom     string `json:"aesrandom"`
	InitSerial    byte   `json:"initserial"`
	TimeStamp     int64  `json:"timestamp"`
	WorkWindow    int64  `json:"workwindow"`
	LinkHeartbeat int64  `json:"linkheartbeat"`
}

func (login *Login) checkLoginReqData(reqMsg *message.Message) (bool, byte) {
	thingDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return false, LoginResultCodeBidErrOrUnreg
	}

	var thingserialno, thingid string
	var status uint8
	err = thingDB.QueryRow("SELECT thingserialno,thingid,status FROM thingbaseinfodata_tbl WHERE bid = ?", reqMsg.MesHeader.Bid).Scan(
		&thingserialno,
		&thingid,
		&status)
	if err != nil {
		logger.Error(err)
		return false, LoginResultCodeBidErrOrUnreg
	}

	//Check status
	if status == ThingUnRegister {
		return false, LoginResultCodeBidErrOrUnreg
	}

	//SN ThingID
	if login.loginReqServData.ThingId != thingid || login.loginReqServData.ThingSN != thingserialno {
		return false, LoginResultCodeSnVinErr
	}

	return true, LoginResultCodeSuccess
}

func (login *Login) genMd5Abstract16Bytes(s string) string {
	data := md5.GenMd5([]byte(s))
	thingRandomMd5 := util.Substr(hex.EncodeToString(data), 0, 16)

	logger.Debug("data =", data, "hex string(data) =", hex.EncodeToString(data))
	logger.Debug("thingRandomMd5 =", thingRandomMd5)

	return thingRandomMd5
}

func (login *Login) saveNewAesKeyAndThingStatus(aesRandom string, bid, eventCreationTime uint32) error {
	thingDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	var prethingaes128key, thingaes128key string
	err = thingDB.QueryRow("SELECT prethingaes128key,thingaes128key FROM thingbaseinfodata_tbl WHERE bid = ?", bid).Scan(
		&prethingaes128key,
		&thingaes128key)
	if err != nil {
		logger.Error(err)
		return err
	}

	var key string
	if login.loginReqServData.KeyType == KeyTypePreAesKey {
		key = prethingaes128key
	} else {
		key = thingaes128key
	}

	newKey := login.genMd5Abstract16Bytes(aesRandom + key)

	stmtUpd, err := thingDB.Prepare("UPDATE thingbaseinfodata_tbl SET status=?,thingaes128key=?,eventcreationtime=? WHERE bid=?")
	if err != nil {
		logger.Error(err)
		return err
	}
	defer stmtUpd.Close()

	_, err = stmtUpd.Exec(ThingRegisteredLogined, newKey, eventCreationTime, bid)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func (login *Login) getAesKeyByKeyType(bid uint32, keyType uint8) (string, error) {
	thingDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	var prethingaes128key, thingaes128key string
	err = thingDB.QueryRow("SELECT prethingaes128key,thingaes128key FROM thingbaseinfodata_tbl WHERE bid = ?", bid).Scan(
		&prethingaes128key,
		&thingaes128key)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	if keyType == KeyTypeCurrentAesKey {
		return thingaes128key, nil
	}

	return prethingaes128key, nil
}

func (login *Login) LoginRequest(thing *Thing, reqMsg *message.Message) error {
	if login.loginStatus != LoginStop {
		logger.Error("Login already start!")
		return errors.New("Login already start!")
	}

	login.loginStatus = LoginRequestStatus

	login.loginReqServData = &LoginReqServData{}
	err := json.Unmarshal(reqMsg.ServData, login.loginReqServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("login.loginReqServData =", string(reqMsg.ServData))

	thing.PushEventChannel(EventLoginChallenge, reqMsg)
	return nil
}

func (login *Login) LoginChallenge(thing *Thing, reqMsg *message.Message) error {
	if login.loginStatus != LoginRequestStatus {
		logger.Error("Need LoginRequest!")
		return errors.New("Need LoginRequest!")
	}

	login.loginStatus = LoginChallengeStatus

	var result byte
	var key string

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	login.loginChallServData = &LoginChallengeServData{}

	//Check data validity
	ok, result := login.checkLoginReqData(reqMsg)
	if ok {
		thingDB, err := database.GetDB(DBName)
		if err != nil {
			logger.Error(err)
			return err
		}

		var prethingaes128key, thingaes128key string
		err = thingDB.QueryRow("SELECT prethingaes128key,thingaes128key FROM thingbaseinfodata_tbl WHERE bid = ?", reqMsg.MesHeader.Bid).Scan(
			&prethingaes128key,
			&thingaes128key)
		if err != nil {
			logger.Error(err)
			return err
		}

		if login.loginReqServData.KeyType == KeyTypePreAesKey {
			key = prethingaes128key
		} else {
			key = thingaes128key
		}

		login.loginChallServData.PlatRandom = util.GenRandomString(16)
		login.loginChallServData.ThingRandomMd5 = login.genMd5Abstract16Bytes(login.loginReqServData.ThingRandom + key + login.loginChallServData.PlatRandom)
	} else {
		login.loginChallServData.PlatRandom = ""
		login.loginChallServData.ThingRandomMd5 = ""
	}

	serviceData, err := json.Marshal(login.loginChallServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("serviceData =", serviceData)
	logger.Debug("serviceDataJson =", string(serviceData))

	/*	aesKey, err := login.getAesKeyByKeyType(respMsg.MesHeader.Bid, login.loginReqServData.KeyType)
		if err != nil {
			logger.Error(err)
			return err
		}
	*/
	encryptServData, err := msg.EncryptServiceData(message.Encrypt_AES128, key, serviceData)
	if err != nil {
		logger.Error(err)
		return err
	}

	//Dispatch data
	dd := message.DispatchData{
		EventCreationTime:    reqMsg.DisPatch.EventCreationTime,
		Aid:                  0x2,
		Mid:                  0x2,
		MessageCounter:       reqMsg.DisPatch.MessageCounter + 1,
		ServiceDataLength:    uint16(len(encryptServData)),
		Result:               result,
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
		MessageFlag:      0x0,
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

	logger.Debug("Send LoginChallenge Success---")

	login.timeoutTimer = time.NewTimer(TimeoutTime)

	if login.closeTimeoutTimer == nil {
		login.closeTimeoutTimer = make(chan bool, 1)
	}

	go func() {
		logger.Debug("Start timer......")
		select {
		case <-login.timeoutTimer.C:
			logger.Warn("Timeout timer coming, login fail!")
			//thing.PushEventChannel(RegisterReqEventMessage, nil)
			login.loginStatus = LoginStop
		case <-login.closeTimeoutTimer:
			logger.Debug("Close Timeout timer!")
		}

		logger.Debug("Timer Close......")
	}()

	return nil
}

func (login *Login) LoginResponse(thing *Thing, respMsg *message.Message) error {
	if login.loginStatus != LoginChallengeStatus {
		logger.Error("Need LoginChallenge!")
		thing.PushEventChannel(EventLoginFailure, respMsg)
		return errors.New("Need LoginChallenge!")
	}

	login.timeoutTimer.Stop()
	login.closeTimeoutTimer <- true
	login.loginStatus = LoginResponseStatus

	thingDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	var prethingaes128key, thingaes128key string
	err = thingDB.QueryRow("SELECT prethingaes128key,thingaes128key FROM thingbaseinfodata_tbl WHERE bid = ?", respMsg.MesHeader.Bid).Scan(
		&prethingaes128key,
		&thingaes128key)
	if err != nil {
		logger.Error(err)
		return err
	}

	var key string
	if login.loginReqServData.KeyType == KeyTypePreAesKey {
		key = prethingaes128key
	} else {
		key = thingaes128key
	}

	login.loginRespServData = &LoginResponseServData{}
	err = json.Unmarshal(respMsg.ServData, login.loginRespServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("login.loginRespServData =", string(respMsg.ServData))

	if login.loginRespServData.AccessKey == login.genMd5Abstract16Bytes(login.loginChallServData.PlatRandom+key) {
		thing.PushEventChannel(EventLoginSuccess, respMsg)
	} else {
		thing.PushEventChannel(EventLoginFailure, respMsg)
	}

	return nil
}

func (login *Login) LoginFailure(thing *Thing, respMsg *message.Message) error {
	if login.loginStatus != LoginResponseStatus {
		logger.Error("Need LoginResponse!")
		//return errors.New("Need LoginResponse!")
	}

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	loginFailServData := &LoginFailureServData{}
	loginFailServData.FailureCode = FailureCodeKeyErr

	var result byte = LoginResultCodeAbstractErr
	if thing.CheckAesKeyOutOfDate(respMsg.MesHeader.Bid) {
		result = LoginResultCodeAesOutOfDate
		loginFailServData.FailureCode = FailureCodeKeyOutOfDate
	}

	if login.loginStatus == LoginStop {
		result = LoginResultCodeInterrupt
	}

	serviceData, err := json.Marshal(loginFailServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("serviceData =", serviceData)
	logger.Debug("serviceDataJson =", string(serviceData))

	var aesKey string
	if login.loginStatus == LoginResponseStatus {
		aesKey, err = login.getAesKeyByKeyType(respMsg.MesHeader.Bid, login.loginReqServData.KeyType)
		if err != nil {
			logger.Error(err)
			return err
		}
	} else {
		aesKey, err = login.getAesKeyByKeyType(respMsg.MesHeader.Bid, KeyTypePreAesKey)
		if err != nil {
			logger.Error(err)
			return err
		}
	}

	encryptServData, err := msg.EncryptServiceData(message.Encrypt_AES128, aesKey, serviceData)
	if err != nil {
		logger.Error(err)
		return err
	}

	//Dispatch data
	dd := message.DispatchData{
		EventCreationTime:    respMsg.DisPatch.EventCreationTime,
		Aid:                  0x2,
		Mid:                  0x4,
		MessageCounter:       respMsg.DisPatch.MessageCounter + 1,
		ServiceDataLength:    uint16(len(encryptServData)),
		Result:               result,
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
		MessageFlag:      0x0,
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

	logger.Debug("Send LoginFailure Success---")
	login.loginStatus = LoginStop
	return nil
}

func (login *Login) LoginSuccess(thing *Thing, respMsg *message.Message) error {
	if login.loginStatus != LoginResponseStatus {
		logger.Error("Need LoginResponse!")
		return errors.New("Need LoginResponse!")
	}

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	loginSuccessServData := &LoginSuccessServData{
		AesRandom:     util.GenRandomString(16),
		InitSerial:    0,
		TimeStamp:     time.Now().Unix(),
		WorkWindow:    time.Now().Unix() + AesKeyOutOfDateTime,
		LinkHeartbeat: 0,
	}

	serviceData, err := json.Marshal(loginSuccessServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("serviceData =", serviceData)
	logger.Debug("serviceDataJson =", string(serviceData))

	aesKey, err := login.getAesKeyByKeyType(respMsg.MesHeader.Bid, login.loginReqServData.KeyType)
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
		Aid:                  0x2,
		Mid:                  0x5,
		MessageCounter:       respMsg.DisPatch.MessageCounter + 1,
		ServiceDataLength:    uint16(len(encryptServData)),
		Result:               LoginResultCodeSuccess,
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
		MessageFlag:      0x0,
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

	logger.Debug("Send LoginSuccess Success---")

	err = login.saveNewAesKeyAndThingStatus(loginSuccessServData.AesRandom, respMsg.MesHeader.Bid, respMsg.DisPatch.EventCreationTime)
	if err != nil {
		logger.Error(err)
		return err
	}

	err = thing.SetThingIdAndBid(login.loginReqServData.ThingId, respMsg.MesHeader.Bid)
	if err != nil {
		logger.Error(err)
		return err
	}

	login.loginStatus = LoginStop
	return nil
}
