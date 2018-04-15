package kafka

import (
	"encoding/binary"
	"errors"
	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
	"github.com/harveywangdao/road/cache/redis"
	"github.com/harveywangdao/road/crypto/aes"
	"github.com/harveywangdao/road/log/logger"
	"github.com/harveywangdao/road/util"
	/*"strconv"*/
	"sync"
	"time"
)

func Consumer1(addrs []string) error {
	var wg sync.WaitGroup

	consumer, err := sarama.NewConsumer(addrs, nil)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer consumer.Close()

	partitionList, err := consumer.Partitions("test")
	if err != nil {
		logger.Error(err)
		return err
	}

	for partition := range partitionList {
		cp, err := consumer.ConsumePartition("test", int32(partition), sarama.OffsetNewest)
		if err != nil {
			logger.Error("partition =", partition, err)
			return err
		}
		defer cp.AsyncClose()

		wg.Add(1)

		go func(pac sarama.PartitionConsumer) {
			defer wg.Done()
			logger.Info("Waiting for consuming message.")
			for msg := range pac.Messages() {
				logger.Info("Partition =", msg.Partition, "Offset =", msg.Offset, "Key =", string(msg.Key), "Value =", string(msg.Value))
			}
		}(cp)
	}

	wg.Wait()

	logger.Info("Done consuming topic test")

	return nil
}

func Consumer2(addrs []string, groupID string, topics []string) error {
	config := cluster.NewConfig()
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true
	config.Consumer.Offsets.CommitInterval = 1 * time.Second
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumer, err := cluster.NewConsumer(addrs, groupID, topics, config)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer consumer.Close()

	go func(consumer *cluster.Consumer) {
		errors := consumer.Errors()
		notifications := consumer.Notifications()
		for {
			select {
			case err := <-errors:
				if err != nil {
					logger.Error(err)
				}
			case ntf := <-notifications:
				logger.Info("ntf =", ntf)
			}
		}
	}(consumer)

	for msg := range consumer.Messages() {
		logger.Info(msg.Topic, msg.Partition, msg.Offset, msg.Key, msg.Value)
		consumer.MarkOffset(msg, "")
	}

	return nil
}

func SyncProducer(addrs []string, topic string) error {
	config := sarama.NewConfig()
	//  config.Producer.RequiredAcks = sarama.WaitForAll
	//  config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.Return.Successes = true
	config.Producer.Timeout = 5 * time.Second

	syncProducer, err := sarama.NewSyncProducer(addrs, config)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer syncProducer.Close()

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder("SyncProducer Value"),
		Key:   sarama.StringEncoder("SyncProducerKey"),
	}

	partition, offset, err := syncProducer.SendMessage(msg)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("partition =", partition, "offset =", offset)
	logger.Info("SyncProducer send message successfully.")

	return nil
}

func AsyncProducer(addrs []string, topic string) error {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Timeout = 5 * time.Second

	asyncProducer, err := sarama.NewAsyncProducer(addrs, config)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer asyncProducer.Close()

	go func(producer sarama.AsyncProducer) {
		errors := producer.Errors()
		success := producer.Successes()
		for {
			select {
			case err := <-errors:
				if err != nil {
					logger.Error(err)
				}
			case <-success:
			}
		}
	}(asyncProducer)

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder("AsyncProducer Value"),
		Key:   sarama.StringEncoder("AsyncProducerKey"),
	}

	asyncProducer.Input() <- msg

	return nil
}

type KafkaMq struct {
}

const (
	MqMessageHeaderId      = 0x12345678
	MqMessageAesEncryptKey = "1234567890123456"
	CrcLen                 = 4
)

type MqMessageHeader struct {
	MqMessageHeaderID uint32
	DataLength        uint32
}

func (ka *KafkaMq) Register() error {
	return nil
}

func PackData(data []byte) ([]byte, error) {
	return data, nil
	packedData := make([]byte, 0, 2048)

	//Encrypt data
	encryptData, err := aes.AesEncrypt(data, []byte(MqMessageAesEncryptKey))
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	//Header
	mqHeader := &MqMessageHeader{
		MqMessageHeaderID: MqMessageHeaderId,
		DataLength:        uint32(len(encryptData)),
	}

	mqHeaderData, err := util.StructToByteSlice(mqHeader)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	packedData = append(packedData, mqHeaderData...)

	//data
	packedData = append(packedData, encryptData...)

	//crc
	crc, err := util.Crc32(packedData)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	crcData, err := util.Uint32ToByteSlice(crc)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	packedData = append(packedData, crcData...)

	return packedData, nil
}

func UnpackData(packedData []byte) ([]byte, error) {
	return packedData, nil
	//Check Crc
	crc, err := util.Crc32(packedData[0 : len(packedData)-CrcLen])
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	packedDataCrc, err := util.ByteSliceToUint32(packedData[len(packedData)-CrcLen:])
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	if crc != packedDataCrc {
		return nil, errors.New("Data Crc Fail!")
	}

	//Check Header
	mqHeader := MqMessageHeader{}
	MqMessageHeaderLen := binary.Size(mqHeader)
	if err = util.ByteSliceToStruct(packedData[0:MqMessageHeaderLen], &mqHeader); err != nil {
		logger.Error(err)
		return nil, err
	}

	if mqHeader.MqMessageHeaderID != MqMessageHeaderId {
		return nil, errors.New("Data Message Header Fail!")
	}

	//Decrypt data
	data, err := aes.AesDecrypt(packedData[MqMessageHeaderLen:len(packedData)-CrcLen], []byte(MqMessageAesEncryptKey))
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return data, nil
}

//////////////////////////////////////////////////Producer///////////////////////////////////////////////////////

type Producer struct {
	AsyncProducer sarama.AsyncProducer
	Addrs         []string
	Topic         string
	done          chan bool
}

func NewProducer(addrs []string, topic string) (*Producer, error) {
	p := &Producer{}
	p.Addrs = addrs
	p.Topic = topic
	p.done = make(chan bool)

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	/*config.Producer.Timeout = 5 * time.Second*/

	var err error
	p.AsyncProducer, err = sarama.NewAsyncProducer(p.Addrs, config)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	/*defer producer.Close()*/

	go func(producer sarama.AsyncProducer, done chan bool) {
		errors := producer.Errors()
		success := producer.Successes()
		var errnum, successnum int = 0, 0

		for {
			select {
			case err, ok := <-errors:
				if !ok {
					logger.Info("Exit from errors and success")
					return
				}
				errnum++
				logger.Info("err =", err, "errnum =", errnum)

			case suc, ok := <-success:
				if !ok {
					logger.Info("Exit from errors and success")
					return
				}
				successnum++
				logger.Debug("Partition =", suc.Partition, "Offset =", suc.Offset) //, "Key =", suc.Key, "Value =", suc.Value)

			case <-done:
				logger.Info("Exit from errors and success")
				return
			}
		}
	}(p.AsyncProducer, p.done)

	return p, nil
}

func (p *Producer) Send(partition int32, data []byte) error {
	logger.Debug("Send data =", data)

	packedData, err := PackData(data)
	if err != nil {
		logger.Error(err)
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: p.Topic,
		Value: sarama.ByteEncoder(packedData),
		//Key:       sarama.StringEncoder("AsyncProducer from TSP Key" + strconv.Itoa(int(partition))),
		Partition: partition,
	}

	p.AsyncProducer.Input() <- msg

	return nil
}

func (p *Producer) Close() {
	close(p.done)
	p.AsyncProducer.Close()
}

//////////////////////////////////////////////////Consumer///////////////////////////////////////////////////////

type Consumer struct {
	ConsumerHigh sarama.Consumer
	Addrs        []string
	Topic        string

	partitionList            []int32
	partitionConsumer        []sarama.PartitionConsumer
	partitionsOffsetRedisKey string
}

const (
	RedisIpPort = "localhost:6379"
)

func NewConsumer(addrs []string, topic string) (*Consumer, error) {
	consumer := &Consumer{}
	consumer.Addrs = addrs
	consumer.Topic = topic
	consumer.partitionConsumer = make([]sarama.PartitionConsumer, 0)
	consumer.partitionsOffsetRedisKey = topic + "OffsetRedisKey"

	var err error
	consumer.ConsumerHigh, err = sarama.NewConsumer(consumer.Addrs, nil)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	/*defer consumer.Close()*/

	consumer.partitionList, err = consumer.ConsumerHigh.Partitions(consumer.Topic)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	logger.Debug("partitionList =", consumer.partitionList)

	red, err := redis.NewRedis(RedisIpPort)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	defer red.Close()

	var offsets []int64 = nil
	//red.DeleteKey(consumer.partitionsOffsetRedisKey)
	if red.IsKeyExist(consumer.partitionsOffsetRedisKey) {
		offsets, err = red.GetInt64SliceList(consumer.partitionsOffsetRedisKey)
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	} else {
		offsets = make([]int64, len(consumer.partitionList))
		for i := 0; i < len(offsets); i++ {
			offsets[i] = -1
		}

		err = red.CreateListByInt64Slice(consumer.partitionsOffsetRedisKey, offsets)
		if err != nil {
			logger.Error(err)
			return nil, err
		}
	}

	for i := 0; i < len(consumer.partitionList); i++ {
		logger.Debug("Offset =", offsets[i])

		pc, err := consumer.ConsumerHigh.ConsumePartition(consumer.Topic, consumer.partitionList[i], offsets[i]+1) //sarama.OffsetNewest)//
		if err != nil {
			logger.Error(err)
			return nil, err
		}
		//defer pc.Close()

		consumer.partitionConsumer = append(consumer.partitionConsumer, pc)
	}

	return consumer, nil
}

func (c *Consumer) Partitions() ([]int32, error) {
	partList := make([]int32, len(c.partitionList))
	copy(partList, c.partitionList)

	return partList, nil
}

func (c *Consumer) Read(partition int32) ([]byte, error) {
	if partition < 0 || partition >= int32(len(c.partitionConsumer)) {
		return nil, errors.New("Partition error.")
	}

	msg := <-c.partitionConsumer[partition].Messages()

	data, err := UnpackData(msg.Value)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	logger.Debug("Partition =", msg.Partition, "Offset =", msg.Offset) //, "Key =", string(msg.Key), string(msg.Value))
	logger.Debug("Read data =", string(data))

	red, err := redis.NewRedis(RedisIpPort)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	defer red.Close()

	err = red.SetListValueByIndex(c.partitionsOffsetRedisKey, int(msg.Partition), msg.Offset)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return data, nil
}

func (c *Consumer) Close() {
	for i := 0; i < len(c.partitionConsumer); i++ {
		c.partitionConsumer[i].Close()
	}

	c.ConsumerHigh.Close()
}
