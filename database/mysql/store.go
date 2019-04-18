package store

import (
	"bms/logger"
	"bms/stat"
	"bms/util"
	"encoding/json"
	"errors"
	"github.com/jinzhu/gorm"
	"os"
	"strings"
	"time"
)

type NetRate struct {
	Device    string  `json:"device"`
	NetRxRate float64 `json:"net_rx_rate"`
	NetTxRate float64 `json:"net_tx_rate"`
}

type IORate struct {
	DiskName    string  `json:"disk_name"`
	IOUsageRate float64 `json:"io_usage_rate"`
}

type MemoryInfo struct {
	MemTotal uint64  `json:"mem_total"`
	MemUsed  uint64  `json:"mem_used"`
	MemFree  uint64  `json:"mem_free"`
	MemRate  float64 `json:"mem_rate"`
}

type Store struct {
	DBDriverName string
	DBHostIP     string
	DBUsername   string
	DBPassword   string
	DBName       string

	firstTime              bool
	lastCPUStat            stat.CPUStat
	lastNetStat            []stat.NetStat
	lastIOStat             []stat.IOStat
	serverIP               string
	saveDataPeriodTime     int
	metricsReservingPeriod int
	metricsCleanupTime     string
}

type SystemInfor struct {
	//gorm.Model
	ID       uint   `gorm:"primary_key"`
	ServerIP string `gorm:"unique_index" json:"server_ip"`
	//ServerIP      string `json:"server_ip"`
	OSInfo        string `json:"os_info"`
	ModelName     string `json:"model_name"`
	CoreCount     int    `json:"core_count"`
	MemoryTotal   uint64 `json:"memory_total"`
	DiskTotal     uint64 `json:"disk_total"`
	DockerVersion string `json:"docker_version"`
	UpdateTime    int64  `gorm:"column:update_time" json:"update_time"`
}

type Metric struct {
	//gorm.Model
	ID int64 `gorm:"primary_key"`
	//CreatedAt time.Time
	//UpdatedAt time.Time
	//DeletedAt *time.Time `sql:"index"`

	ServerIP string `gorm:"size:20;index" json:"server_ip"`
	Time     int64  `gorm:"column:timestamp;index" json:"timestamp"`

	CPURate float64 `json:"cpu_rate"`

	//MemInfo string `gorm:"type:varchar(128)" json:"mem_info"`
	MemRate float64 `json:"mem_rate"`

	DiskInfo string `gorm:"type:varchar(512)" json:"disk_info"`

	IOUsageRate string `gorm:"type:varchar(512)" json:"io_usage_rate"`

	NetInfo string `gorm:"type:varchar(1024)" json:"net_info"`
}

func (s *Store) getCPURate(cpuStat *stat.CPUStat) float64 {
	var rate float64

	lastCPUTotalTime := s.lastCPUStat.User + s.lastCPUStat.Nice + s.lastCPUStat.System + s.lastCPUStat.Idle + s.lastCPUStat.IOWait + s.lastCPUStat.IRQ + s.lastCPUStat.SoftIRQS + s.lastCPUStat.Steal + s.lastCPUStat.Guest + s.lastCPUStat.GuestNice

	currentCPUTotalTime := cpuStat.User + cpuStat.Nice + cpuStat.System + cpuStat.Idle + cpuStat.IOWait + cpuStat.IRQ + cpuStat.SoftIRQS + cpuStat.Steal + cpuStat.Guest + cpuStat.GuestNice

	totalTimePeriod := float64(currentCPUTotalTime - lastCPUTotalTime)
	idlePeriod := float64(cpuStat.Idle - s.lastCPUStat.Idle)

	logger.Debug("currentCPUTotalTime =", currentCPUTotalTime, "lastCPUTotalTime =", lastCPUTotalTime)
	logger.Debug("cpuStat.Idle =", cpuStat.Idle, "s.lastCPUStat.Idle =", s.lastCPUStat.Idle)
	logger.Debug("totalTimePeriod =", totalTimePeriod, "idlePeriod =", idlePeriod)
	logger.Debug("idlePeriod/totalTimePeriod =", idlePeriod/totalTimePeriod)

	rate = (1.00 - idlePeriod/totalTimePeriod) * 100.00

	rate = util.Round(rate, 2)

	logger.Debug("rate =", rate)
	logger.Debug("idle =", 100-rate)

	return rate
}

func (s *Store) getNetRate(currentNet []stat.NetStat) {
	netRates := []NetRate{}

	if len(currentNet) != len(s.lastNetStat) {
		logger.Error("network interface number is error!")
		return
	}

	var period int = 1 //1 second

	for i := 0; i < len(currentNet); i++ {
		if currentNet[i].Device != s.lastNetStat[i].Device {
			logger.Error("network interface name error!")
			return
		}
		netRate := NetRate{}
		netRate.Device = currentNet[i].Device

		rxKB := float64(currentNet[i].RXBytes-s.lastNetStat[i].RXBytes) / 1024.00
		txKB := float64(currentNet[i].TXBytes-s.lastNetStat[i].TXBytes) / 1024.00

		netRate.NetRxRate = rxKB / float64(period) //KB/S
		netRate.NetTxRate = txKB / float64(period) //KB/S

		netRate.NetRxRate = util.Round(netRate.NetRxRate, 2)
		netRate.NetTxRate = util.Round(netRate.NetTxRate, 2)

		netRates = append(netRates, netRate)
	}

	logger.Debug("netRates =", netRates)
}

func (s *Store) getIORate(ioStat []stat.IOStat, diskStat []stat.DiskStat) ([]IORate, error) {
	ioRates := []IORate{}

	if len(ioStat) != len(s.lastIOStat) {
		logger.Error("Disk number changed")
		return nil, errors.New("Disk number changed")
	}

	for i, _ := range ioStat {
		if ioStat[i].Name != s.lastIOStat[i].Name {
			logger.Error("Disk number changed")
			return nil, errors.New("Disk number changed")
		}
	}

	for _, disk := range diskStat {
		for i, _ := range ioStat {
			if ioStat[i].Name == strings.TrimPrefix(disk.Label, "/dev/") {
				ioRate := IORate{}

				ioRate.DiskName = ioStat[i].Name
				ioRate.IOUsageRate = float64(ioStat[i].IOTicks-s.lastIOStat[i].IOTicks) / float64(ioStat[i].TimeStamp-s.lastIOStat[i].TimeStamp) * 100.00
				ioRate.IOUsageRate = util.Round(ioRate.IOUsageRate, 2)
				logger.Debug("ioRate.IOUsageRate =", ioRate.IOUsageRate)
				ioRates = append(ioRates, ioRate)
				break
			}
		}
	}

	logger.Debug("ioRates =", ioRates)

	return ioRates, nil
}

func (s *Store) SaveSysInfo() error {
	dbParam := s.DBUsername + ":" + s.DBPassword + "@tcp(" + s.DBHostIP + ")/" + s.DBName + "?charset=utf8&parseTime=True&loc=Local"
	db, err := gorm.Open(s.DBDriverName, dbParam)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer db.Close()

	//db.DropTableIfExists(&Metric{})
	//db.DropTableIfExists(&SystemInfor{})

	if !db.HasTable(&SystemInfor{}) {
		db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&SystemInfor{})
	} else {
		//db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").AutoMigrate(&SystemInfor{})
	}

	//host
	host, err := stat.GetHostInfo()
	if err != nil {
		logger.Error(err)
		return err
	}

	//CPU info
	info, err := stat.GetCPUInfo()
	if err != nil {
		logger.Error(err)
		return err
	}

	//memory
	mem, err := stat.GetMemStat()
	if err != nil {
		logger.Error(err)
		return err
	}

	//disk
	ds, err := stat.GetDiskStat()
	if err != nil {
		logger.Error(err)
		return err
	}

	var DiskTotal uint64 = 0

	for _, v := range ds {
		DiskTotal += v.Total
	}

	//docker version
	version, _ := GetDockerVersion()

	db = db.Exec("UPDATE system_infors SET os_info=?,model_name=?,core_count=?,memory_total=?,disk_total=?,docker_version=?,update_time=? WHERE server_ip=?",
		host.OSType+" "+host.OSRelease+" "+host.Release+" "+host.OSBit,
		info.ModelName,
		info.CoreCount,
		mem.MemTotal,
		DiskTotal,
		version,
		time.Now().Unix(),
		s.serverIP)
	if db.Error != nil {
		logger.Error(err)
		return err
	}

	if db.RowsAffected == 0 {
		err = db.Exec("INSERT INTO system_infors(server_ip,os_info,model_name,core_count,memory_total,disk_total,docker_version,update_time) VALUES(?,?,?,?,?,?,?,?)",
			s.serverIP,
			host.OSType+" "+host.OSRelease+" "+host.Release+" "+host.OSBit,
			info.ModelName,
			info.CoreCount,
			mem.MemTotal,
			DiskTotal,
			version,
			time.Now().Unix()).Error
		if err != nil {
			logger.Error(err)
			return err
		}
	}

	/*err = db.Exec("INSERT INTO system_infors(server_ip,os_info,model_name,core_count,memory_total,disk_total,docker_version) "+
		"VALUES(?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE "+
		"os_info = VALUES(os_info),"+
		"model_name = VALUES(model_name),"+
		"core_count = VALUES(core_count),"+
		"memory_total = VALUES(memory_total),"+
		"disk_total = VALUES(disk_total),"+
		"docker_version = VALUES(docker_version)",
		s.serverIP,
		host.OSType+" "+host.OSRelease+" "+host.Release+" "+host.OSBit,
		info.ModelName,
		info.CoreCount,
		mem.MemTotal,
		DiskTotal,
		version).Error
	if err != nil {
		logger.Error(err)
		return err
	}*/

	return nil
}

func (s *Store) initMetricsData() error {
	cpuSt, err := stat.GetCPUStat()
	if err != nil {
		logger.Error(err)
		return err
	}
	s.lastCPUStat = cpuSt

	netSt, err := stat.GetNetStat()
	if err != nil {
		logger.Error(err)
		return err
	}
	s.lastNetStat = netSt

	ioSt, err := stat.GetIOStat()
	if err != nil {
		logger.Error(err)
		return err
	}
	s.lastIOStat = ioSt

	return nil
}

func (s *Store) SaveMetrics() error {
	metric := Metric{
		Time:     time.Now().Unix(),
		ServerIP: s.serverIP,
	}

	//CPU info
	st, err := stat.GetCPUStat()
	if err != nil {
		logger.Error(err)
		return err
	}

	metric.CPURate = s.getCPURate(&st)
	s.lastCPUStat = st

	//memory
	mem, err := stat.GetMemStat()
	if err != nil {
		logger.Error(err)
		return err
	}
	logger.Debug(mem)

	metric.MemRate = mem.MemRate

	//disk
	ds, err := stat.GetDiskStat()
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("disk stat =", ds)

	diskStatJson, err := json.Marshal(ds)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("diskStatJson =", string(diskStatJson))
	metric.DiskInfo = string(diskStatJson)

	// io
	io, err := stat.GetIOStat()
	if err != nil {
		logger.Error(err)
		return err
	}
	logger.Debug(io)

	ioRate, err := s.getIORate(io, ds)
	if err != nil {
		logger.Error(err)
		return err
	}

	ioRateJson, err := json.Marshal(ioRate)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("ioRateJson =", string(ioRateJson))
	metric.IOUsageRate = string(ioRateJson)

	s.lastIOStat = io

	//net
	net, err := stat.GetNetStat()
	if err != nil {
		logger.Error(err)
		return err
	}
	logger.Debug(net)

	netStatJson, err := json.Marshal(net)
	if err != nil {
		logger.Error(err)
		return err
	}
	metric.NetInfo = string(netStatJson)
	logger.Debug("metric.NetInfo =", string(metric.NetInfo))

	//s.getNetRate(net)
	s.lastNetStat = net

	/*
		//load
		load, err := stat.GetLoadStat()
		if err != nil {
			logger.Error(err)
			return err
		}
		logger.Debug(load)

		//uptime
		uptime, err := stat.GetUptimeStat()
		if err != nil {
			logger.Error(err)
			return err
		}
		logger.Debug(uptime)
	*/

	dbParam := s.DBUsername + ":" + s.DBPassword + "@tcp(" + s.DBHostIP + ")/" + s.DBName + "?charset=utf8&parseTime=True&loc=Local"
	db, err := gorm.Open(s.DBDriverName, dbParam)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer db.Close()

	if !db.HasTable(&Metric{}) {
		db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&Metric{})
	} else {
		//db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").AutoMigrate(&Metric{})
	}

	err = db.Create(&metric).Error
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func (s *Store) storeTask() {
	defer func() {
		logger.Info("Exit from Save Data Ticker")
	}()

	saveMetricsTicker := time.NewTicker(time.Duration(s.saveDataPeriodTime) * time.Millisecond)

	for {
		select {
		case <-saveMetricsTicker.C:
			logger.Debug("Save metrics data to DB")
			s.SaveMetrics()
		}
	}
}

func (s *Store) getMetricsCleanupTime(timePoint string) (int64, error) {
	ymd := time.Now().Format("2006-01-02")

	loc, err := time.LoadLocation("Local")
	if err != nil {
		logger.Error(err)
		return -1, err
	}

	cleanupTime, err := time.ParseInLocation("2006-01-02 15:04:05", ymd+" "+timePoint, loc)
	if err != nil {
		logger.Error("timePoint =", timePoint, err)
		return -1, err
	}

	return cleanupTime.Unix(), nil
}

func (s *Store) deleteMetricsTask() {
	defer func() {
		logger.Info("Exit from deleteMetricsTask")
	}()

	var oneDaySecondNumber int64 = 24 * 60 * 60
	cleanupTime, err := s.getMetricsCleanupTime(s.metricsCleanupTime)
	if err != nil {
		logger.Error(err)
		os.Exit(1)
		return
	}

	nowTime := time.Now().Unix()
	if nowTime >= cleanupTime {
		cleanupTime += oneDaySecondNumber //跨一天
	}

	deleteMetricsTimer := time.NewTimer(time.Second * time.Duration(cleanupTime-nowTime))

	for {
		select {
		case <-deleteMetricsTimer.C:
			nowTime = time.Now().Unix()
			cleanupTime += oneDaySecondNumber //跨一天

			deleteMetricsTimer.Reset(time.Second * time.Duration(cleanupTime-nowTime))

			dbParam := s.DBUsername + ":" + s.DBPassword + "@tcp(" + s.DBHostIP + ")/" + s.DBName + "?charset=utf8&parseTime=True&loc=Local"
			db, err := gorm.Open(s.DBDriverName, dbParam)
			if err != nil {
				logger.Error(err)
				break
			}

			deleteTime := nowTime - int64(s.metricsReservingPeriod*24*60*60)
			logger.Info("Start delete metrics data from DB before", deleteTime)

			//DELETE FROM metrics WHERE server_ip=s.serverIP AND timestamp<deleteTime ORDER BY timestamp LIMIT 10000
			err = db.Delete(Metric{}, "server_ip=? AND timestamp<?", s.serverIP, deleteTime).Error
			if err != nil {
				logger.Error(err)
				db.Close()
				break
			}

			db.Close()
			logger.Info("Success delete metrics data from DB before", deleteTime)
		}
	}
}

func (s *Store) storeTask2() {
	defer func() {
		logger.Info("Exit from Save Data Ticker")
	}()

	dbParam := s.DBUsername + ":" + s.DBPassword + "@tcp(" + s.DBHostIP + ")/" + s.DBName + "?charset=utf8&parseTime=True&loc=Local"
	db, err := gorm.Open(s.DBDriverName, dbParam)
	if err != nil {
		logger.Error(err)
		return
	}
	defer db.Close()

	for {
		metric := Metric{
			Time:     time.Now().Unix(),
			ServerIP: s.serverIP,
		}

		metric.CPURate = 0.1254
		metric.MemRate = 15.71
		metric.DiskInfo = `[{"label":"/dev/sda1","used":33457928,"free":47993824}]`
		metric.IOUsageRate = `[{"disk_name":"sda1","io_usage_rate":0.4}]`
		metric.NetInfo = `[{"device":"ens33","rx_bytes":227914746,"tx_bytes":484488491}]`
		err = db.Create(&metric).Error
		if err != nil {
			logger.Error(err)
			//return err
		}
	}
}

func NewStore(ip, dbDriverName, dbHostIP, dbUsername, dbPassword, dbName string, saveDataPeriodTime, metricsReservingPeriod int, metricsCleanupTime string) (*Store, error) {
	store := new(Store)
	store.firstTime = true

	store.DBDriverName = dbDriverName
	store.DBHostIP = dbHostIP
	store.DBUsername = dbUsername
	store.DBPassword = dbPassword
	store.DBName = dbName
	store.serverIP = ip
	store.saveDataPeriodTime = saveDataPeriodTime
	store.metricsReservingPeriod = metricsReservingPeriod
	store.metricsCleanupTime = metricsCleanupTime

	if store.serverIP == "" {
		logger.Error("IP error")
		return nil, errors.New("IP error")
	}

	store.SaveSysInfo()
	store.initMetricsData()

	go store.storeTask()
	go store.deleteMetricsTask()
	//go store.storeTask2()

	logger.Info("start storing system data")

	return store, nil
}
