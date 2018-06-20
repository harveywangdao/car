package main

import (
	"github.com/harveywangdao/road/iot/server"
	"github.com/harveywangdao/road/log/logger"
	"log"
)

func initIoT() {
	//fileHandler := logger.NewFileHandler("test.log")
	//logger.SetHandlers(logger.Console, fileHandler)
	logger.SetHandlers(logger.Console)
	//defer logger.Close()
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetLevel(logger.INFO)
}

func main() {
	initIoT()
	logger.Debug("Start Server...")
	server.Server()
}
