package main

import (
	_ "github.com/harveywangdao/road/database"
	"github.com/harveywangdao/road/iot_client/client"
	"github.com/harveywangdao/road/log/logger"
	"log"
)

func initIotClient() {
	//fileHandler := logger.NewFileHandler("test.log")
	//logger.SetHandlers(logger.Console, fileHandler)
	logger.SetHandlers(logger.Console)
	//defer logger.Close()
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetLevel(logger.INFO)
}

func main() {
	initIotClient()
	logger.Debug("Start Client...")
	client.Client()
}
