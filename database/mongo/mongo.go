package mongo

import (
	"errors"
	"gopkg.in/mgo.v2"
	/*	"gopkg.in/mgo.v2/bson"*/
	"github.com/harveywangdao/road/log/logger"
	"sync"
)

const (
	MgoAddr = "localhost:27017" //IP地址
)

var mgos map[string]*mgo.Session
var lock sync.Mutex

func init() {
	mgos = make(map[string]*mgo.Session)
}

type Database struct {
}

func openMongo(ipport string) (*mgo.Session, error) {
	session, err := mgo.Dial(ipport)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	//defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	session.SetPoolLimit(64)

	return session, nil
}

func Close(mgoAddr string) error {
	session, ok := mgos[mgoAddr]
	if !ok {
		return errors.New(mgoAddr + " not exist!")
	}

	lock.Lock()
	defer lock.Unlock()

	session, ok = mgos[mgoAddr]
	if !ok {
		return errors.New(mgoAddr + " not exist!")
	}

	session.Close()
	delete(mgos, mgoAddr)

	return nil
}

func CloneMgoSession() (*mgo.Session, error) {
	mgoAddr := MgoAddr
	session, ok := mgos[mgoAddr]
	if ok {
		return session.Clone(), nil
	}

	lock.Lock()
	defer lock.Unlock()

	session, ok = mgos[mgoAddr]
	if ok {
		return session.Clone(), nil
	}

	session, err := openMongo(mgoAddr)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	mgos[mgoAddr] = session

	return session.Clone(), nil
}
