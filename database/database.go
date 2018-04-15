package database

import (
	"database/sql"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"github.com/harveywangdao/road/log/logger"
	"sync"
)

const (
	DBHostIP   = "127.0.0.1:3306" //IP地址
	DBUsername = "root"           //用户名
	DBPassword = "180498"         //密码
	DriverName = "mysql"
)

var dbMap map[string]*sql.DB
var lock sync.Mutex

func init() {
	dbMap = make(map[string]*sql.DB)
}

type Database struct {
}

func openDB(dbName string) (*sql.DB, error) {
	dataSourceName := DBUsername + ":" + DBPassword + "@tcp(" + DBHostIP + ")/" + dbName + "?charset=utf8"
	logger.Debug("dataSourceName =", dataSourceName)

	db, err := sql.Open(DriverName, dataSourceName)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	//defer db.Close()

	/*	err = db.Ping()
		if err != nil {
			db.Close()
			logger.Error(err)
			return nil, err
		}*/

	return db, nil
}

func Close(dbname string) error {
	db, ok := dbMap[dbname]
	if !ok {
		return errors.New(dbname + " not exist!")
	}

	lock.Lock()
	defer lock.Unlock()

	db, ok = dbMap[dbname]
	if !ok {
		return errors.New(dbname + " not exist!")
	}

	err := db.Close()
	if err != nil {
		logger.Error(err)
		return err
	}

	delete(dbMap, dbname)

	return nil
}

func GetDB(dbname string) (*sql.DB, error) {
	db, ok := dbMap[dbname]
	if ok {
		return db, nil
	}

	lock.Lock()
	defer lock.Unlock()

	db, ok = dbMap[dbname]
	if ok {
		return db, nil
	}

	db, err := openDB(dbname)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	dbMap[dbname] = db

	return db, nil
}
