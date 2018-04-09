package tbox

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"hcxy/iov/crypto/md5"
	"hcxy/iov/database"
	"hcxy/iov/log/logger"
	"hcxy/iov/message"
	"hcxy/iov/util"
	"time"
)

const (
	TimeoutTime = 1 * time.Second

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

	loginStatus int
}

type LoginReqServData struct {
	KeyType    byte   `json:"keytype"`
	TboxSN     string `json:"tboxsn"`
	Vin        string `json:"vin"`
	TboxRandom string `json:"tboxrandom"`
}

type LoginChallengeServData struct {
	TboxRandomMd5 string `json:"tboxrandommd5"`
	PlatRandom    string `json:"platrandom"`
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

func (login *Login) getAesKeyByKeyType(tboxNo int, keyType uint8) (string, error) {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	var pretboxaes128key, tboxaes128key string
	err = tboxDB.QueryRow("SELECT pretboxaes128key,tboxaes128key FROM tboxfactorydata_tbl WHERE id = ?", tboxNo).Scan(
		&pretboxaes128key,
		&tboxaes128key)
	if err != nil {
		logger.Error(err)
		return "", err
	}

	if keyType == KeyTypeCurrentAesKey {
		return tboxaes128key, nil
	}

	return pretboxaes128key, nil
}

func (login *Login) genMd5Abstract16Bytes(s string) string {
	data := md5.GenMd5([]byte(s))
	tboxRandomMd5 := util.Substr(hex.EncodeToString(data), 0, 16)

	logger.Debug("data =", data, "hex string(data) =", hex.EncodeToString(data))
	logger.Debug("tboxRandomMd5 =", tboxRandomMd5)

	return tboxRandomMd5
}

func (login *Login) saveNewAesKey(aesRandom string, eventCreationTime uint32, tboxNo int) error {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	var pretboxaes128key, tboxaes128key string
	err = tboxDB.QueryRow("SELECT pretboxaes128key,tboxaes128key FROM tboxfactorydata_tbl WHERE id = ?", tboxNo).Scan(
		&pretboxaes128key,
		&tboxaes128key)
	if err != nil {
		logger.Error(err)
		return err
	}

	var key string
	if login.loginReqServData.KeyType == KeyTypePreAesKey {
		key = pretboxaes128key
	} else {
		key = tboxaes128key
	}

	newKey := login.genMd5Abstract16Bytes(aesRandom + key)

	stmtUpd, err := tboxDB.Prepare("UPDATE tboxfactorydata_tbl SET tboxaes128key=?,eventcreationtime=? WHERE id=?")
	if err != nil {
		logger.Error(err)
		return err
	}
	defer stmtUpd.Close()

	_, err = stmtUpd.Exec(newKey, eventCreationTime, tboxNo)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func (login *Login) checkAesKeyOutOfDate(tboxNo int) bool {
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return false
	}

	var eventcreationtime uint32
	err = tboxDB.QueryRow("SELECT eventcreationtime FROM tboxfactorydata_tbl WHERE id = ?", tboxNo).Scan(
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

func (login *Login) LoginRequest(eve *Event) error {
	if login.loginStatus != LoginStop {
		logger.Error("Login already start!")
		return errors.New("Login already start!")
	}

	if eve.TboxStatus == TboxUnRegister {
		logger.Error("Tbox UnRegister!")
		return errors.New("Tbox UnRegister!")
	}

	login.loginStatus = LoginRequestStatus

	msg := message.Message{
		Connection: eve.Conn,
	}

	//Service data
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	var tboxserialno, vin string
	var bid uint32
	err = tboxDB.QueryRow("SELECT tboxserialno,vin,bid FROM tboxfactorydata_tbl WHERE id = ?", eve.TboxNo).Scan(
		&tboxserialno,
		&vin,
		&bid)
	if err != nil {
		logger.Error(err)
		return err
	}

	var keyType byte
	if login.checkAesKeyOutOfDate(eve.TboxNo) {
		keyType = KeyTypePreAesKey
	} else {
		keyType = KeyTypeCurrentAesKey
	}

	login.loginReqServData = &LoginReqServData{
		KeyType:    keyType, /*0-pre key; 1-current key*/
		TboxSN:     tboxserialno,
		Vin:        vin,
		TboxRandom: util.GenRandomString(16),
	}

	serviceData, err := json.Marshal(login.loginReqServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("serviceData =", serviceData)
	logger.Debug("serviceDataJson =", string(serviceData))

	aesKey, err := login.getAesKeyByKeyType(eve.TboxNo, keyType)
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

	logger.Info("Send LoginRequest success......")

	login.timeoutTimer = time.NewTimer(TimeoutTime)

	if login.closeTimeoutTimer == nil {
		login.closeTimeoutTimer = make(chan bool, 1)
	}

	go func() {
		logger.Info("Start timer......")
		select {
		case <-login.timeoutTimer.C:
			logger.Warn("Timeout timer coming, login fail, register again!")
			eve.PushEventChannel(RegisterReqEventMessage, nil)
			login.loginStatus = LoginStop
		case <-login.closeTimeoutTimer:
			logger.Info("Close Timeout timer!")
		}

		logger.Info("Timer Close......")
	}()

	return nil
}

func (login *Login) LoginChallenge(eve *Event, challengeMsg *message.Message) error {
	if login.loginStatus != LoginRequestStatus {
		logger.Error("Need LoginRequest!")
		return errors.New("Need LoginRequest!")
	}

	login.loginStatus = LoginChallengeStatus

	login.timeoutTimer.Stop()

	if challengeMsg.DisPatch.Result != LoginResultCodeSuccess {
		logger.Info("Close timer")
		login.closeTimeoutTimer <- true
		eve.PushEventChannel(RegisterReqEventMessage, nil)
		login.loginStatus = LoginStop
		return nil
	}

	login.loginChallServData = &LoginChallengeServData{}
	err := json.Unmarshal(challengeMsg.ServData, login.loginChallServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("login.loginChallServData =", string(challengeMsg.ServData))

	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	var pretboxaes128key, tboxaes128key string
	err = tboxDB.QueryRow("SELECT pretboxaes128key,tboxaes128key FROM tboxfactorydata_tbl WHERE id = ?", eve.TboxNo).Scan(
		&pretboxaes128key,
		&tboxaes128key)
	if err != nil {
		logger.Error(err)
		return err
	}

	var key string
	if login.loginReqServData.KeyType == KeyTypePreAesKey {
		key = pretboxaes128key
	} else {
		key = tboxaes128key
	}

	localTboxRandomMd5 := login.genMd5Abstract16Bytes(login.loginReqServData.TboxRandom + key + login.loginChallServData.PlatRandom)

	if login.loginChallServData.TboxRandomMd5 != localTboxRandomMd5 {
		logger.Info("Close timer")
		login.closeTimeoutTimer <- true
		eve.PushEventChannel(RegisterReqEventMessage, nil)
		login.loginStatus = LoginStop
	} else {
		eve.PushEventChannel(EventLoginResponse, challengeMsg)
	}

	return nil
}

func (login *Login) LoginResponse(eve *Event, challengeMsg *message.Message) error {
	if login.loginStatus != LoginChallengeStatus {
		logger.Error("Need LoginChallenge!")
		return errors.New("Need LoginChallenge!")
	}

	login.loginStatus = LoginResponseStatus

	msg := message.Message{
		Connection: eve.Conn,
	}

	//Service data
	tboxDB, err := database.GetDB(DBName)
	if err != nil {
		logger.Error(err)
		return err
	}

	var pretboxaes128key, tboxaes128key string
	err = tboxDB.QueryRow("SELECT pretboxaes128key,tboxaes128key FROM tboxfactorydata_tbl WHERE id = ?", eve.TboxNo).Scan(
		&pretboxaes128key,
		&tboxaes128key)
	if err != nil {
		logger.Error(err)
		return err
	}

	var key string
	if login.loginReqServData.KeyType == KeyTypePreAesKey {
		key = pretboxaes128key
	} else {
		key = tboxaes128key
	}

	login.loginRespServData = &LoginResponseServData{
		SerialUP:  util.GenRandomString(16),
		AccessKey: login.genMd5Abstract16Bytes(login.loginChallServData.PlatRandom + key),
	}

	serviceData, err := json.Marshal(login.loginRespServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("serviceData =", serviceData)
	logger.Debug("serviceDataJson =", string(serviceData))

	/*	aesKey, err := login.getAesKeyByKeyType(eve.TboxNo, keyType)
		if err != nil {
			logger.Error(err)
			return err
		}*/

	encryptServData, err := msg.EncryptServiceData(message.Encrypt_AES128, key, serviceData)
	if err != nil {
		logger.Error(err)
		return err
	}

	//Dispatch data
	dd := message.DispatchData{
		EventCreationTime:    challengeMsg.DisPatch.EventCreationTime,
		Aid:                  0x2,
		Mid:                  0x3,
		MessageCounter:       challengeMsg.DisPatch.MessageCounter + 1,
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

	//Message header
	mh := message.MessageHeader{
		FixHeader:        message.MessageHeaderID,
		ServiceDataCheck: util.DataXOR(serviceData),
		ServiceVersion:   0x0, //not sure
		Bid:              challengeMsg.MesHeader.Bid,
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

	logger.Debug("Send LoginResponse success......")

	login.timeoutTimer.Reset(TimeoutTime)
	return nil
}

func (login *Login) LoginFailure(eve *Event, failMsg *message.Message) error {
	if login.loginStatus != LoginResponseStatus {
		logger.Error("Need LoginResponse!")
		return errors.New("Need LoginResponse!")
	}

	login.timeoutTimer.Stop()
	logger.Info("Close timer")
	login.closeTimeoutTimer <- true

	if failMsg.DisPatch.Result == LoginResultCodeInterrupt {
		logger.Error("Login again!")
		eve.PushEventChannel(EventLoginRequest, nil)
	} else {
		logger.Error("Register again!")
		eve.PushEventChannel(RegisterReqEventMessage, nil)
	}

	login.loginStatus = LoginStop

	return nil
}

func (login *Login) LoginSuccess(eve *Event, successMsg *message.Message) error {
	if login.loginStatus != LoginResponseStatus {
		logger.Error("Need LoginResponse!")
		return errors.New("Need LoginResponse!")
	}

	login.timeoutTimer.Stop()
	logger.Info("Close timer")
	login.closeTimeoutTimer <- true

	loginSuccessServData := &LoginSuccessServData{}
	err := json.Unmarshal(successMsg.ServData, loginSuccessServData)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("loginSuccessServData =", string(successMsg.ServData))

	err = login.saveNewAesKey(loginSuccessServData.AesRandom, successMsg.DisPatch.EventCreationTime, eve.TboxNo)
	if err != nil {
		logger.Error(err)
		return err
	}

	eve.TboxStatus = TboxRegisteredLogined
	eve.PushEventChannel(EventSaveTboxStatusLogined, nil)

	login.loginStatus = LoginStop

	return nil
}
