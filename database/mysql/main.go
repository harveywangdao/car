package main

import (
	"bms/httpserver"
	"bms/logger"
	"bms/store"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	//"fmt"
	"github.com/go-ini/ini"
	"log"
	"os"
	"strings"
	//"runtime"
	"sync"
)

var (
	BMSConfig   *ini.File
	logDirPath  = "log"
	logFilePath = "log/test.log"
)

func initConfig() {
	var err error
	BMSConfig, err = ini.Load("conf/my.ini")
	if err != nil {
		logger.Error(err)
		os.Exit(1)
		return
	}
}

func createDBTable(dbDriverName, dbHostIP, dbUsername, dbPassword, dbName string) {
	dbParam := dbUsername + ":" + dbPassword + "@tcp(" + dbHostIP + ")/" + dbName + "?charset=utf8&parseTime=True&loc=Local"
	db, err := gorm.Open(dbDriverName, dbParam)
	if err != nil {
		logger.Error(err)
		os.Exit(1)
		return
	}
	defer db.Close()

	if !db.HasTable(&store.Metric{}) {
		err = db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&store.Metric{}).Error
		if err != nil {
			logger.Error(err)
			os.Exit(1)
			return
		}
		logger.Info("Create Metric table")
	} else {
		//db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").AutoMigrate(&Metric{})
	}

	if !db.HasTable(&store.SystemInfor{}) {
		err = db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&store.SystemInfor{}).Error
		if err != nil {
			logger.Error(err)
			os.Exit(1)
			return
		}
		logger.Info("Create SystemInfor table")
	}

	if !db.HasTable(&store.ApiServerStatus{}) {
		err = db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&store.ApiServerStatus{}).Error
		if err != nil {
			logger.Error(err)
			os.Exit(1)
			return
		}
		logger.Info("Create ApiServerStatus table")
	}

	if !db.HasTable(&store.ContainerAndPeer{}) {
		err = db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&store.ContainerAndPeer{}).Error
		if err != nil {
			logger.Error(err)
			os.Exit(1)
			return
		}
		logger.Info("Create ContainerAndPeer table")
	}

	if !db.HasTable(&store.IpAccessList{}) {
		err = db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&store.IpAccessList{}).Error
		if err != nil {
			logger.Error(err)
			os.Exit(1)
			return
		}
		logger.Info("Create IpAccessList table")
	}

	if !db.HasTable(&store.User{}) {
		err = db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&store.User{}).Error
		if err != nil {
			logger.Error(err)
			os.Exit(1)
			return
		}
		logger.Info("Create User table")
	}

	if !db.HasTable(&store.EarlyRadar{}) {
		err = db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&store.EarlyRadar{}).Error
		if err != nil {
			logger.Error(err)
			os.Exit(1)
			return
		}
		logger.Info("Create EarlyRadar table")
	}

	if !db.HasTable(&store.Vcode{}) {
		err = db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&store.Vcode{}).Error
		if err != nil {
			logger.Error(err)
			os.Exit(1)
			return
		}
		logger.Info("Create Vcode table")
	}
}

func init() {
	//fmt.Println("Blockchain Monitoring System initialize...")
	//runtime.GOMAXPROCS(runtime.NumCPU())
	st, err := os.Stat(logDirPath)
	if err == nil {
		if !st.IsDir() {
			log.Fatal(logDirPath, "is not dir")
		}
	} else {
		if os.IsNotExist(err) {
			err = os.Mkdir(logDirPath, os.ModePerm)
			if err != nil {
				log.Fatal("mkdir fail")
			}
		} else {
			log.Fatal(logDirPath, "error")
		}
	}

	fileHandler := logger.NewFileHandler(logFilePath)
	logger.SetHandlers(logger.Console, fileHandler)
	//defer logger.Close()
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetLevel(logger.INFO)

	initConfig()
}

func main() {
	var wg sync.WaitGroup

	dbDriverName := BMSConfig.Section("").Key("DBDriverName").String()
	dbHostIP := BMSConfig.Section("").Key("DBHostIP").String()
	dbUsername := BMSConfig.Section("").Key("DBUsername").String()
	dbPassword := BMSConfig.Section("").Key("DBPassword").String()
	dbName := BMSConfig.Section("").Key("DBName").String()

	createDBTable(dbDriverName, dbHostIP, dbUsername, dbPassword, dbName)

	ip := BMSConfig.Section("").Key("ServerIP").String()
	saveDataPeriodTime, _ := BMSConfig.Section("").Key("FreshPeriodTime").Int()
	metricsReservingPeriod, _ := BMSConfig.Section("").Key("MetricsReservingPeriod").Int()
	metricsCleanupTime := BMSConfig.Section("").Key("MetricsCleanupTime").String()

	_, err := store.NewStore(ip, dbDriverName, dbHostIP, dbUsername, dbPassword, dbName, saveDataPeriodTime, metricsReservingPeriod, metricsCleanupTime)
	if err != nil {
		logger.Error(err)
		return
	}

	moniterPeer, _ := BMSConfig.Section("").Key("MoniterPeer").Bool()
	if moniterPeer {
		peerInfoMap := make(map[string]store.PeerInfo)
		location := BMSConfig.Section("").Key("Location").String()
		latitude := BMSConfig.Section("").Key("LocationLatitude").String()
		longitude := BMSConfig.Section("").Key("LocationLongitude").String()

		keys := BMSConfig.Section("PeerIpPort").Keys()
		for _, v := range keys {
			peerInfo := store.PeerInfo{}
			peerInfo.PeerIp = strings.Trim(strings.Split(v.Value(), ",")[0], " ")
			peerInfo.PeerName = strings.Trim(strings.Split(v.Value(), ",")[1], " ")
			peerInfoMap[v.Name()] = peerInfo
		}

		_, err = store.NewDockerMonitor(ip, dbDriverName, dbHostIP, dbUsername, dbPassword, dbName, saveDataPeriodTime, peerInfoMap, location, latitude, longitude)
		if err != nil {
			logger.Error(err)
			return
		}
	}

	nginx, _ := BMSConfig.Section("").Key("Nginx").Bool()
	if nginx {
		nginxLog := BMSConfig.Section("").Key("NginxLog").String()
		backupNginxLogTime := BMSConfig.Section("").Key("NginxLogBackupTime").String()
		_, err = store.NewMonitorNginx(dbDriverName, dbHostIP, dbUsername, dbPassword, dbName, nginxLog, backupNginxLogTime)
		if err != nil {
			logger.Error(err)
			return
		}
	}

	api, _ := BMSConfig.Section("").Key("ApiServer").Bool()
	if api {
		apiip := BMSConfig.Section("").Key("ApiServerIp").String()
		apiport := BMSConfig.Section("").Key("ApiServerPort").String()
		period, _ := BMSConfig.Section("").Key("ApiServerCheckPeriod").Int()
		_, err = store.NewApiServer(ip, dbDriverName, dbHostIP, dbUsername, dbPassword, dbName, apiip, apiport, period)
		if err != nil {
			logger.Error(err)
			return
		}
	}

	server, _ := BMSConfig.Section("").Key("HttpServer").Bool()
	if server {
		port := BMSConfig.Section("").Key("HttpServerPort").String()
		_, err = httpserver.NewHttpServer(port)
		if err != nil {
			logger.Error(err)
			return
		}
	}

	wg.Add(1)
	wg.Wait()
	logger.Info("Exit from BMS")
}
