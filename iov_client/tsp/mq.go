package tsp

import (
	"hcxy/iov/log/logger"
	"hcxy/iov/msgqueue"
	"sync"
)

const (
	ProducerRoutineNum = 20
	ConsumerRoutineNum = 1
	MQAddr             = "localhost:9092"
	SendMessageTopic   = "TspToIov"
	RecvMessageTopic   = "IovToTsp"
)

func MQ(mqToTspChan, tspToMqChan chan []byte) {
	var wg sync.WaitGroup
	wg.Add(1)
	mqs, err := msgqueue.NewMqService(MQAddr, RecvMessageTopic, SendMessageTopic, ConsumerRoutineNum, ProducerRoutineNum, mqToTspChan, tspToMqChan)
	if err != nil {
		logger.Error(err)
		return
	}

	mqs.Start()

	wg.Wait()
}
