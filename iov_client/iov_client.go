package main

import (
	_ "hcxy/iov/database"
	"hcxy/iov/iov_client/client"
	"hcxy/iov/log/logger"
	"log"
)

func initIovClient() {
	fileHandler := logger.NewFileHandler("test.log")
	logger.SetHandlers(logger.Console, fileHandler)
	//defer logger.Close()
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetLevel(logger.INFO)
}

func main() {
	initIovClient()
	logger.Debug("Start Client...")
	client.Client()
}
