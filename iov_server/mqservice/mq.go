package mqservice

import (
	"hcxy/iov/log/logger"
	"hcxy/iov/msgqueue"
	"sync"
)

const (
	ConsumerRoutineNum = 1
	ProducerRoutineNum = 10
	MQAddr             = "localhost:9092"
	RecvMessageTopic   = "TspToIov"
	SendMessageTopic   = "IovToTsp"
)

func MQ(mqToTboxChan, tboxToMqChan chan []byte) {
	var wg sync.WaitGroup
	wg.Add(1)
	mqs, err := msgqueue.NewMqService(MQAddr, RecvMessageTopic, SendMessageTopic, ConsumerRoutineNum, ProducerRoutineNum, mqToTboxChan, tboxToMqChan)
	if err != nil {
		logger.Error(err)
		return
	}

	mqs.Start()

	wg.Wait()
}
