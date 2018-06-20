package thing

import (
	"github.com/harveywangdao/road/log/logger"
	"sync"
)

const (
	maxGoroutineNum = 1
	IPPort          = "127.0.0.1:6024"
)

func thingConnHandler(wg sync.WaitGroup, thingNo int) {
	defer wg.Done()

	thingMsgChan := make(chan ThingMessage, 128)

	thing, err := NewThing(thingMsgChan, IPPort, thingNo)
	if err != nil {
		logger.Error(err)
		return
	}

	err = thing.ThingScheduler()
	if err != nil {
		logger.Error(err)
		return
	}
}

func Things() {
	var wg sync.WaitGroup

	for i := 0; i < maxGoroutineNum; i++ {
		wg.Add(1)
		go thingConnHandler(wg, i+1)
	}

	wg.Wait()
}
