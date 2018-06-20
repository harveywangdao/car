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
	done := make(chan bool)

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	/*config.Producer.Timeout = 5 * time.Second*/

	logger.Info("Connect kafka...")

	asyncProducer, err := sarama.NewAsyncProducer(addrs, config)
	if err != nil {
		logger.Error(err)
		return
	}
	defer asyncProducer.Close()

	logger.Info("Connect kafka success!")
	var wg sync.WaitGroup
	wg.Add(1)
	go func(producer sarama.AsyncProducer, done chan bool) {
		errors := producer.Errors()
		success := producer.Successes()
		var errnum, successnum int = 0, 0

		for {
			select {
			case err, ok := <-errors:
				if !ok {
					logger.Error("Exit from errors.")
					return
				}
				errnum++
				logger.Error("Send kafka fail!", "Topic =", err.Msg.Topic, "err =", err, "errnum =", errnum)

			case suc, ok := <-success:
				if !ok {
					logger.Error("Exit from success")
					return
				}
				successnum++
				logger.Info("Send kafka success!", "Topic =", suc.Topic, "Partition =", suc.Partition, "Offset =", suc.Offset) //, "Key =", suc.Key, "Value =", suc.Value)

			case <-done:
				logger.Error("Exit")
				return
			}
		}
	}(asyncProducer, done)

	data := "ncjaksdcnsalknclkamnlk"
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(data),
		//Key:       sarama.StringEncoder("AsyncProducer from TSP Key" + strconv.Itoa(int(partition))),
		Partition: -1,
	}

	logger.Info("Send kafka msg =", string(data))
	asyncProducer.Input() <- msg
	logger.Info("Sending kafka...")
	wg.Wait()
}
