package tsp

import (
	"hcxy/iov/eventmsg"
	"hcxy/iov/log/logger"
	"hcxy/iov/util"
	"net"
	"sync"
)

const (
	Port = ":1236"
)

type TboxIdMsgChanMap struct {
	TboxId          uint32
	EventMsgChannel chan *eventmsg.EventMsgChan
}

var (
	tidMsgChMapList []*TboxIdMsgChanMap
	lock            sync.Mutex
)

func genTboxId(tboxIdMapList []*TboxIdMsgChanMap) uint32 {
	if len(tboxIdMapList) == 0 {
		return 0x12345651
	}

	return tboxIdMapList[len(tboxIdMapList)-1].TboxId + 1
}

func AddTid(eventMsgChannel chan *eventmsg.EventMsgChan) {
	lock.Lock()
	defer lock.Unlock()

	tid := &TboxIdMsgChanMap{
		TboxId:          genTboxId(tidMsgChMapList),
		EventMsgChannel: eventMsgChannel,
	}

	tidMsgChMapList = append(tidMsgChMapList, tid)
}

func DeleteTid(tid uint32) {
	lock.Lock()
	defer lock.Unlock()

	var i int
	for i = 0; i < len(tidMsgChMapList); i++ {
		if tidMsgChMapList[i].TboxId == tid {
			break
		}
	}

	if i >= len(tidMsgChMapList) {
		logger.Error("Can not find the tid:", tid)
		return
	}

	tidMsgChMapList = append(tidMsgChMapList[:i], tidMsgChMapList[i+1:]...)
}

func GetMsgChannel(tid uint32) chan *eventmsg.EventMsgChan {
	lock.Lock()
	defer lock.Unlock()

	for _, t := range tidMsgChMapList {
		if t.TboxId == tid {
			return t.EventMsgChannel
		}
	}

	return nil
}

func AddTidHeader(tid uint32, data []byte) ([]byte, error) {
	tidHeaderData, err := util.Uint32ToByteSlice(tid)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	tidHeaderData = append(tidHeaderData, data...)

	return tidHeaderData, nil
}

func GetTidHeader(data []byte) uint32 {
	tid, err := util.ByteSliceToUint32(data[:4])
	if err != nil {
		logger.Error(err)
		return 0
	}

	return tid
}

func connectionHandler(conn net.Conn, tidMap *TboxIdMsgChanMap, mqToTspChan, tspToMqChan chan []byte) {
	defer conn.Close()
	logger.Debug("Net =", conn.LocalAddr().Network(), ", Addr =", conn.LocalAddr().String())
	logger.Debug("Remote net =", conn.RemoteAddr().Network(), ", Remote addr =", conn.RemoteAddr().String())

	te, err := NewTboxEvent(conn, tidMap.EventMsgChannel, tidMap.TboxId, tspToMqChan, mqToTspChan)
	if err != nil {
		logger.Error(err)
		return
	}

	err = te.Start()
	if err != nil {
		logger.Error(err)
		return
	}
}

func openListener(mqToTspChan, tspToMqChan chan []byte) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", Port)
	if err != nil {
		logger.Error(err)
		return
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		logger.Error(err)
		return
	}
	defer listener.Close()

	logger.Debug("Net =", listener.Addr().Network(), ", Addr =", listener.Addr().String())

	for {
		logger.Info("TSP waiting connecting...")
		conn, err := listener.Accept()
		if err != nil {
			logger.Error(err)
			return
		}

		eventMsgChannel := make(chan *eventmsg.EventMsgChan, 128)

		lock.Lock()

		tidMap := &TboxIdMsgChanMap{
			TboxId:          genTboxId(tidMsgChMapList),
			EventMsgChannel: eventMsgChannel,
		}

		tidMsgChMapList = append(tidMsgChMapList, tidMap)
		go connectionHandler(conn, tidMap, mqToTspChan, tspToMqChan)

		lock.Unlock()
	}
}

func Tsp() {
	var wg sync.WaitGroup

	mqToTspChan := make(chan []byte, 128)
	tspToMqChan := make(chan []byte, 128)
	tidMsgChMapList = make([]*TboxIdMsgChanMap, 0, 128)

	wg.Add(1)
	go openListener(mqToTspChan, tspToMqChan)
	go MQ(mqToTspChan, tspToMqChan)

	recvMq := RecvMq{
		RecvMqChan: mqToTspChan,
		SendMqChan: tspToMqChan,
	}

	err := recvMq.Start()
	if err != nil {
		logger.Error(err)
	}

	wg.Wait()
}
