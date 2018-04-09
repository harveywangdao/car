package main

import (
	"hcxy/iov/iov_server/server"
	"hcxy/iov/log/logger"
	"log"
)

func initIOV() {
	fileHandler := logger.NewFileHandler("test.log")
	logger.SetHandlers(logger.Console, fileHandler)
	//defer logger.Close()
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetLevel(logger.INFO)
}

func main() {
	initIOV()
	logger.Debug("Start Server...")
	server.Server()
}
