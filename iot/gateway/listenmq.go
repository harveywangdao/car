package gateway

import (
	"github.com/harveywangdao/road/log/logger"
	"github.com/harveywangdao/road/msgqueue"
)

type ListenMQ struct {
	ConsumerRoutineNum int
	ProducerRoutineNum int
	MQAddr             string
	RecvMessageTopic   string
	SendMessageTopic   string
	MqToGatewayChan    chan []byte
	GatewayToMqChan    chan []byte
}

func (listen *ListenMQ) Run() {
	mqs, err := msgqueue.NewMqService(listen.MQAddr, listen.RecvMessageTopic, listen.SendMessageTopic, listen.ConsumerRoutineNum, listen.ProducerRoutineNum, listen.MqToGatewayChan, listen.GatewayToMqChan)
	if err != nil {
		logger.Error(err)
		return
	}

	mqs.Start()
}
