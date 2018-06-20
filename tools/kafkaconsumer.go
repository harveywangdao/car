package main

import (
	"github.com/Shopify/sarama"
	"hcxy/iov/log/logger"
	"log"
	"sync"
)

func main() {
	logger.SetHandlers(logger.Console)
	//defer logger.Close()
	logger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger.SetLevel(logger.INFO)

	//addrs := []string{"222.177.234.116:9092"}
	addrs := []string{"139.199.4.94:9092"}
	//topic := "testkafka"
	topic := "TboxDataSirunTspToXinyuan"

	ConsumerHigh, err := sarama.NewConsumer(addrs, nil)
	if err != nil {
		logger.Error(err)
		return
	}
	defer ConsumerHigh.Close()

	partitionList, err := ConsumerHigh.Partitions(topic)
	if err != nil {
		logger.Error(err)
		return
	}

	logger.Info("partitionList =", partitionList)
	partitionConsumer := make([]sarama.PartitionConsumer, 0)

	for i := 0; i < len(partitionList); i++ {
		pc, err := ConsumerHigh.ConsumePartition(topic, partitionList[i], sarama.OffsetNewest)
		if err != nil {
			logger.Error(err)
			return
		}
		defer pc.Close()
		partitionConsumer = append(partitionConsumer, pc)
	}

	var wg sync.WaitGroup
	for i := 0; i < len(partitionList); i++ {
		partition := partitionList[i]
		wg.Add(1)

		go func(part int32) {
			defer wg.Done()
			logger.Info("Waiting for consuming message.", "partition =", part)
			for {
				msg := <-partitionConsumer[part].Messages()
				logger.Info("Topic =", msg.Topic, "Partition =", msg.Partition, "Offset =", msg.Offset)
				logger.Info("Recv MQ data =", msg.Value)
			}
		}(int32(partition))
	}

	wg.Wait()
}
