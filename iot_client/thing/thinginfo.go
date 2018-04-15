package thing

import (
	"github.com/harveywangdao/road/log/logger"
	//"github.com/harveywangdao/road/util"
	"time"
)

type ThingInfor struct {
	Version uint32
	Time    uint32

	IsLocation uint8
	Latitude   uint32
	Longitude  uint32
	Heading    uint16
	Speed      uint16
}

func (info *ThingInfor) GetThingInfor() *ThingInfor {
	info.Version = 1235
	info.Time = uint32(time.Now().Unix())
	info.IsLocation = 77
	info.Latitude = 88
	info.Longitude = 99
	info.Heading = 00
	info.Speed = 55
	logger.Debug("ThingInfor =", *info)
	return info
}
