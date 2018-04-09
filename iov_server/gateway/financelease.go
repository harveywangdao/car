package gateway

import (
	"hcxy/iov/log/logger"
	"hcxy/iov/msgqueue"
	"sync"
)

type FinanceLease struct {
	ConsumerRoutineNum int
	ProducerRoutineNum int
	MQAddr             string
	RecvMessageTopic   string
	SendMessageTopic   string
	MqToGatewayChan    chan []byte
	GatewayToMqChan    chan []byte
}

func (fl *FinanceLease) FinanceLeaseStart() error {
	var wg sync.WaitGroup
	wg.Add(1)
	mqs, err := msgqueue.NewMqService(fl.MQAddr, fl.RecvMessageTopic, fl.SendMessageTopic, fl.ConsumerRoutineNum, fl.ProducerRoutineNum, fl.MqToGatewayChan, fl.GatewayToMqChan)
	if err != nil {
		logger.Error(err)
		return err
	}

	mqs.Start()
	wg.Wait()

	return nil
}
