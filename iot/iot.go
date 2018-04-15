package main

import (
	"github.com/harveywangdao/road/iot/server"
	"github.com/harveywangdao/road/log/logger"
	"log"
)

func initIOT() {
	//fileHandler := logger.NewFileHandler("test.log")
	//logger.SetHandlers(logger.Console, fileHandler)
	logger.SetHandlers(logger.Console)
	//defer logger.Close()
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetLevel(logger.INFO)
}

func main() {
	initIOT()
	logger.Debug("Start Server...")
	server.Server()
}
