package tboxservice

import (
	"hcxy/iov/log/logger"
	"net"
	"sync"
)

func TBox(tboxconnToTboxChan chan net.Conn, mqToTboxChan, tboxToMqChan chan []byte) {
	var wg sync.WaitGroup

	wg.Add(1)

	go ConnectListener(tboxconnToTboxChan)

	dispatcher := TboxDispatcher{
		MqToTboxChan: mqToTboxChan,
		TboxToMqChan: tboxToMqChan,
	}

	err := dispatcher.Start()
	if err != nil {
		logger.Error(err)
	}

	wg.Wait()
}
