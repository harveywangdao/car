package thing

import (
	"database/sql"
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
	TimeoutTime    = 10 * time.Second
	LoginAgainTime = 10 * time.Second

	LoginResultCodeSuccess       = 0x00
	LoginResultCodeSnVinErr      = 0xA8
	LoginResultCodeBidErrOrUnreg = 0xA9
	LoginResultCodeInterrupt     = 0xAA
	LoginResultCodeAbstractErr   = 0xAB
	LoginResultCodeAesOutOfDate  = 0xAC

	KeyTypePreAesKey     = 0
	KeyTypeCurrentAesKey = 1
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

	loginStatus         int
	loginEventCreatTime uint32
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

type LoginSuccessServData struct {
	AesRandom     string `json:"aesrandom"`
	InitSerial    byte   `json:"initserial"`
	TimeStamp     int64  `json:"timestamp"`
	WorkWindow    int64  `json:"workwindow"`
	LinkHeartbeat int64  `json:"linkheartbeat"`
}

func (login *Login) getAesKeyByKeyType(thingNo int, keyType uint8) (string, error) {
	thingDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	var prethingaes128key, thingaes128key string
	err = thingDB.QueryRow("SELECT prethingaes128key,thingaes128key FROM thingbaseinfodata_tbl WHERE id = ?", thingNo).Scan(
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

func (login *Login) genMd5Abstract16Bytes(s string) string {
	data := md5.GenMd5([]byte(s))
	thingRandomMd5 := util.Substr(hex.EncodeToString(data), 0, 16)

	logger.Debug("data =", data, "hex string(data) =", hex.EncodeToString(data))
	logger.Debug("thingRandomMd5 =", thingRandomMd5)

	return thingRandomMd5
}

func (login *Login) saveNewAesKey(aesRandom string, eventCreationTime uint32, thingNo int) error {
	thingDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	var prethingaes128key, thingaes128key string
	err = thingDB.QueryRow("SELECT prethingaes128key,thingaes128key FROM thingbaseinfodata_tbl WHERE id = ?", thingNo).Scan(
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

	stmtUpd, err := thingDB.Prepare("UPDATE thingbaseinfodata_tbl SET thingaes128key=?,eventcreationtime=? WHERE id=?")
	if err != nil {
		logger.Error(err)
		return err
	}
	defer stmtUpd.Close()

	_, err = stmtUpd.Exec(newKey, eventCreationTime, thingNo)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func (login *Login) checkAesKeyOutOfDate(thingNo int) bool {
	thingDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return false
	}

	var eventcreationtime uint32
	err = thingDB.QueryRow("SELECT eventcreationtime FROM thingbaseinfodata_tbl WHERE id = ?", thingNo).Scan(
		&eventcreationtime)
	if err != nil {
		logger.Error(err)
		return false
	}

	if time.Now().Unix()-int64(eventcreationtime) >= AesKeyOutOfDateTime {
		return true
	}

	return false
}

func (login *Login) loginRequestSendData(thing *Thing) error {
	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	db, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	var thingserialno, thingid string
	var bid uint32
	err = db.QueryRow("SELECT thingserialno,thingid,bid FROM thingbaseinfodata_tbl WHERE id = ?", thing.ThingNo).Scan(
		&thingserialno,
		&thingid,
		&bid)
	if err != nil {
		logger.Error(err)
		return err
	}

	var keyType byte
	if login.checkAesKeyOutOfDate(thing.ThingNo) {
		keyType = KeyTypePreAesKey
	} else {
		keyType = KeyTypeCurrentAesKey
	}

	login.loginReqServData = &LoginReqServData{
		KeyType:     keyType, /*0-pre key; 1-current key*/
		ThingSN:     thingserialno,
		ThingId:     thingid,
		ThingRandom: util.GenRandomString(16),
	}

	serviceData, err := json.Marshal(login.loginReqServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("serviceData =", serviceData)
	logger.Debug("serviceDataJson =", string(serviceData))

	aesKey, err := login.getAesKeyByKeyType(thing.ThingNo, keyType)
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
		Aid:                  0x2,
		Mid:                  0x1,
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

	login.loginEventCreatTime = dd.EventCreationTime

	//Message header
	mh := message.MessageHeader{
		FixHeader:        message.MessageHeaderID,
		ServiceDataCheck: util.DataXOR(serviceData),
		ServiceVersion:   0x0, //not sure
		Bid:              bid,
		MessageFlag:      0x0,
	}

	messageHeaderData, err := util.StructToByteSlice(mh)
	if err != nil {
		logger.Error(err)
		return err
	}

	err = msg.SendMessage(messageHeaderData, dispatchData, encryptServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func (login *Login) LoginRequest(thing *Thing) error {
	if login.loginStatus != LoginStop {
		logger.Error("Login already start!")
		return errors.New("Login already start!")
	}

	if thing.ThingStatus == ThingUnRegister {
		logger.Error("Thing UnRegister!")
		return errors.New("Thing UnRegister!")
	}

	login.loginStatus = LoginRequestStatus

	err := login.loginRequestSendData(thing)
	if err != nil {
		t := time.NewTimer(LoginAgainTime)

		go func() {
			select {
			case <-t.C:
				logger.Info("Login fail, Login again!")
				thing.PushEventChannel(EventLoginRequest, nil)
			}
		}()

		login.loginStatus = LoginStop

		return err
	}

	logger.Debug("Send LoginRequest success......")

	login.timeoutTimer = time.NewTimer(TimeoutTime)

	if login.closeTimeoutTimer == nil {
		login.closeTimeoutTimer = make(chan bool, 1)
	}

	go func() {
		logger.Debug("Start timer......")
		select {
		case <-login.timeoutTimer.C:
			logger.Warn("Timeout timer coming, login fail, register again!")
			thing.PushEventChannel(RegisterReqEventMessage, nil)
			login.loginStatus = LoginStop
		case <-login.closeTimeoutTimer:
			logger.Debug("Close Timeout timer!")
		}

		logger.Debug("Timer Close......")
	}()

	return nil
}

func (login *Login) LoginChallenge(thing *Thing, challengeMsg *message.Message) error {
	if login.loginStatus != LoginRequestStatus {
		logger.Error("Need LoginRequest!")
		return errors.New("Need LoginRequest!")
	}

	if login.loginEventCreatTime != challengeMsg.DisPatch.EventCreationTime {
		logger.Error("Package out of date!")
		return errors.New("Package out of date!")
	}

	login.loginStatus = LoginChallengeStatus

	login.timeoutTimer.Stop()

	var prethingaes128key, thingaes128key, localThingRandomMd5, key string
	var db *sql.DB
	var err error

	if challengeMsg.DisPatch.Result != LoginResultCodeSuccess {
		logger.Debug("Result error!")
		goto FAILURE
	}

	login.loginChallServData = &LoginChallengeServData{}
	err = json.Unmarshal(challengeMsg.ServData, login.loginChallServData)
	if err != nil {
		logger.Error(err)
		goto FAILURE
	}

	logger.Debug("login.loginChallServData =", string(challengeMsg.ServData))

	db, err = database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		goto FAILURE
	}

	err = db.QueryRow("SELECT prethingaes128key,thingaes128key FROM thingbaseinfodata_tbl WHERE id = ?", thing.ThingNo).Scan(
		&prethingaes128key,
		&thingaes128key)
	if err != nil {
		logger.Error(err)
		goto FAILURE
	}

	if login.loginReqServData.KeyType == KeyTypePreAesKey {
		key = prethingaes128key
	} else {
		key = thingaes128key
	}

	localThingRandomMd5 = login.genMd5Abstract16Bytes(login.loginReqServData.ThingRandom + key + login.loginChallServData.PlatRandom)

	if login.loginChallServData.ThingRandomMd5 != localThingRandomMd5 {
		logger.Error("RandomMd5 error!")
		goto FAILURE
	} else {
		thing.PushEventChannel(EventLoginResponse, challengeMsg)
	}

	return nil

FAILURE:
	logger.Debug("Close timer")
	login.closeTimeoutTimer <- true
	thing.PushEventChannel(RegisterReqEventMessage, nil)
	login.loginStatus = LoginStop

	return err
}

func (login *Login) LoginResponse(thing *Thing, challengeMsg *message.Message) error {
	if login.loginStatus != LoginChallengeStatus {
		logger.Error("Need LoginChallenge!")
		return errors.New("Need LoginChallenge!")
	}

	login.loginStatus = LoginResponseStatus

	msg := message.Message{
		Connection: thing.Conn,
	}

	//Service data
	var prethingaes128key, thingaes128key, key string
	var serviceData, encryptServData, dispatchData, messageHeaderData []byte
	var dd message.DispatchData
	var mh message.MessageHeader

	db, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		goto FAILURE
	}

	err = db.QueryRow("SELECT prethingaes128key,thingaes128key FROM thingbaseinfodata_tbl WHERE id = ?", thing.ThingNo).Scan(
		&prethingaes128key,
		&thingaes128key)
	if err != nil {
		logger.Error(err)
		goto FAILURE
	}

	if login.loginReqServData.KeyType == KeyTypePreAesKey {
		key = prethingaes128key
	} else {
		key = thingaes128key
	}

	login.loginRespServData = &LoginResponseServData{
		SerialUP:  util.GenRandomString(16),
		AccessKey: login.genMd5Abstract16Bytes(login.loginChallServData.PlatRandom + key),
	}

	serviceData, err = json.Marshal(login.loginRespServData)
	if err != nil {
		logger.Error(err)
		goto FAILURE
	}

	logger.Debug("serviceData =", serviceData)
	logger.Debug("serviceDataJson =", string(serviceData))

	/*	aesKey, err := login.getAesKeyByKeyType(thing.ThingNo, keyType)
		if err != nil {
			logger.Error(err)
			return err
		}*/

	encryptServData, err = msg.EncryptServiceData(message.Encrypt_AES128, key, serviceData)
	if err != nil {
		logger.Error(err)
		goto FAILURE
	}

	//Dispatch data
	dd = message.DispatchData{
		EventCreationTime:    challengeMsg.DisPatch.EventCreationTime,
		Aid:                  0x2,
		Mid:                  0x3,
		MessageCounter:       challengeMsg.DisPatch.MessageCounter + 1,
		ServiceDataLength:    uint16(len(encryptServData)),
		Result:               0,
		SecurityVersion:      message.Encrypt_AES128,
		DispatchCreationTime: uint32(time.Now().Unix()),
	}

	dispatchData, err = util.StructToByteSlice(dd)
	if err != nil {
		logger.Error(err)
		goto FAILURE
	}

	//Message header
	mh = message.MessageHeader{
		FixHeader:        message.MessageHeaderID,
		ServiceDataCheck: util.DataXOR(serviceData),
		ServiceVersion:   0x0, //not sure
		Bid:              challengeMsg.MesHeader.Bid,
		MessageFlag:      0x0,
	}

	messageHeaderData, err = util.StructToByteSlice(mh)
	if err != nil {
		logger.Error(err)
		goto FAILURE
	}

	err = msg.SendMessage(messageHeaderData, dispatchData, encryptServData)
	if err != nil {
		logger.Error(err)
		goto FAILURE
	}

	logger.Debug("Send LoginResponse success......")

	login.timeoutTimer.Reset(TimeoutTime)
	return nil

FAILURE:
	logger.Debug("Close timer")
	login.closeTimeoutTimer <- true

	t := time.NewTimer(LoginAgainTime)

	go func() {
		select {
		case <-t.C:
			logger.Info("Login fail, Login again!")
			thing.PushEventChannel(EventLoginRequest, nil)
		}
	}()

	login.loginStatus = LoginStop

	return err
}

func (login *Login) LoginFailure(thing *Thing, failMsg *message.Message) error {
	if login.loginStatus != LoginResponseStatus {
		logger.Error("Need LoginResponse!")
		return errors.New("Need LoginResponse!")
	}

	if login.loginEventCreatTime != failMsg.DisPatch.EventCreationTime {
		logger.Error("Package out of date!")
		return errors.New("Package out of date!")
	}

	login.timeoutTimer.Stop()
	logger.Debug("Close timer")
	login.closeTimeoutTimer <- true

	if failMsg.DisPatch.Result == LoginResultCodeInterrupt {
		logger.Error("Login again!")
		thing.PushEventChannel(EventLoginRequest, nil)
	} else {
		logger.Error("Register again!")
		thing.PushEventChannel(RegisterReqEventMessage, nil)
	}

	login.loginStatus = LoginStop

	return nil
}

func (login *Login) LoginSuccess(thing *Thing, successMsg *message.Message) error {
	if login.loginStatus != LoginResponseStatus {
		logger.Error("Need LoginResponse!")
		return errors.New("Need LoginResponse!")
	}

	if login.loginEventCreatTime != successMsg.DisPatch.EventCreationTime {
		logger.Error("Package out of date!")
		return errors.New("Package out of date!")
	}

	login.timeoutTimer.Stop()
	logger.Debug("Close timer")
	login.closeTimeoutTimer <- true

	loginSuccessServData := &LoginSuccessServData{}
	err := json.Unmarshal(successMsg.ServData, loginSuccessServData)
	if err != nil {
		logger.Error(err)
		login.loginStatus = LoginStop
		thing.PushEventChannel(EventLoginRequest, nil)
		return err
	}

	logger.Debug("loginSuccessServData =", string(successMsg.ServData))

	err = login.saveNewAesKey(loginSuccessServData.AesRandom, successMsg.DisPatch.EventCreationTime, thing.ThingNo)
	if err != nil {
		logger.Error(err)
		login.loginStatus = LoginStop
		thing.PushEventChannel(EventLoginRequest, nil)
		return err
	}

	thing.ThingStatus = ThingRegisteredLogined
	thing.SetThingStatusToDB(ThingRegisteredLogined)
	logger.Info(login.loginReqServData.ThingId, "Login success!")
	login.loginStatus = LoginStop

	return nil
}
