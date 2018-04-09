package tbox

import (
	"hcxy/iov/log/logger"
	"sync"
)

const (
	maxGoroutineNum = 1
	IPPort          = "127.0.0.1:1235"
	//IPPort = "127.0.0.1:1236"
)

func tboxConnHandler(wg sync.WaitGroup, tboxNo int) {
	defer wg.Done()

	eventMsgChan := make(chan EventMessage, 128)

	eve, err := NewEvent(eventMsgChan, IPPort, tboxNo)
	if err != nil {
		logger.Error(err)
		return
	}

	err = eve.EventScheduler()
	if err != nil {
		logger.Error(err)
		return
	}
}

func TBox() {
	var wg sync.WaitGroup

	for i := 0; i < maxGoroutineNum; i++ {
		wg.Add(1)
		go tboxConnHandler(wg, i+1)
	}

	wg.Wait()
}
