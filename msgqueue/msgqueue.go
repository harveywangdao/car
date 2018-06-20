package msgqueue

import (
	"github.com/harveywangdao/road/log/logger"
	"github.com/harveywangdao/road/msgqueue/kafka"
	"sync"
)

type MqService struct {
	MqAddr             string
	RecvTopic          string
	SendTopic          string
	ConsumerRoutineNum int
	ProducerRoutineNum int
	RecvMqChan         chan []byte
	SendMqChan         chan []byte
}

func NewMqService(mqAddr, recvTopic, sendTopic string, consumerRoutineNum, producerRoutineNum int, recvMqChan, sendMqChan chan []byte) (*MqService, error) {
	mqs := &MqService{
		MqAddr:             mqAddr,
		RecvTopic:          recvTopic,
		SendTopic:          sendTopic,
		ConsumerRoutineNum: consumerRoutineNum,
		ProducerRoutineNum: producerRoutineNum,
		RecvMqChan:         recvMqChan,
		SendMqChan:         sendMqChan,
	}

	return mqs, nil
}

func (mqs *MqService) Close() {

}

func (mqs *MqService) Start() {
	var wg sync.WaitGroup
	wg.Add(1)
	go mqs.mqRecvRoutine()
	go mqs.mqSendRoutine()
	wg.Wait()
}

func (mqs *MqService) consumerRoutine(faWg sync.WaitGroup) error {
	defer faWg.Done()
	addrs := []string{mqs.MqAddr}

	consumer, err := kafka.NewConsumer(addrs, mqs.RecvTopic)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer consumer.Close()

	partitionList, err := consumer.Partitions()
	if err != nil {
		logger.Error(err)
		return err
	}

	var wg sync.WaitGroup
	for i := 0; i < len(partitionList); i++ {
		partition := partitionList[i]
		wg.Add(1)

		go func(part int32) {
			defer wg.Done()
			logger.Debug("Waiting for consuming message.", "partition =", part)
			for {
				data, err := consumer.Read(part)
				if err != nil {
					logger.Error(err)
					//return
				}

				logger.Debug("Recv MQ data =", data)
				mqs.RecvMqChan <- data
			}
		}(int32(partition))
	}

	wg.Wait()

	return nil
}

func (mqs *MqService) mqRecvRoutine() {
	var wg sync.WaitGroup
	for i := 0; i < mqs.ConsumerRoutineNum; i++ {
		wg.Add(1)
		go mqs.consumerRoutine(wg)
	}

	wg.Wait()
}

func (mqs *MqService) producerRoutine(faWg sync.WaitGroup) {
	defer faWg.Done()
	addrs := []string{mqs.MqAddr}

	p, err := kafka.NewProducer(addrs, mqs.SendTopic)
	if err != nil {
		logger.Error(err)
		return
	}
	defer p.Close()

	for {
		select {
		case data, ok := <-mqs.SendMqChan:
			if !ok {
				logger.Error("TboxToMqChan exit!")
				return
			}

			logger.Debug("Send MQ data =", data)
			err = p.Send(-1, data)
			if err != nil {
				logger.Error(err)
				return
			}
		}
	}
}

func (mqs *MqService) mqSendRoutine() {
	var wg sync.WaitGroup
	for i := 0; i < mqs.ProducerRoutineNum; i++ {
		wg.Add(1)
		go mqs.producerRoutine(wg)
	}

	wg.Wait()
}
