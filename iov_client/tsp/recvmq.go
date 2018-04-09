package tsp

import (
	"hcxy/iov/eventmsg"
	"hcxy/iov/log/logger"
)

type RecvMq struct {
	RecvMqChan chan []byte
	SendMqChan chan []byte

	mqServiceChan chan []byte
	tboxMqConn    TboxMqConnection
}

type TboxMqConnection struct {
	SendMqChan chan []byte
}

func (tmp TboxMqConnection) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (tmp TboxMqConnection) Write(b []byte) (n int, err error) {
	tmp.SendMqChan <- b
	return len(b), nil
}

func (r *RecvMq) demux(data []byte) error {
	tid := GetTidHeader(data)

	msgChan := GetMsgChannel(tid)

	if msgChan != nil {
		emc := &eventmsg.EventMsgChan{
			Event: EventSendToTbox,
			Param: data,
		}

		msgChan <- emc
	}

	logger.Debug("data =", data)
	return nil
}

func (r *RecvMq) recvFromMq() {
	for {
		data, ok := <-r.RecvMqChan
		if !ok {
			logger.Error("RecvMqChan Close!")
			return
		}

		r.mqServiceChan <- data
	}
}

func (r *RecvMq) Start() error {
	r.mqServiceChan = make(chan []byte, 128)

	go r.recvFromMq()

	for {
		select {
		case data := <-r.mqServiceChan:
			err := r.demux(data)
			if err != nil {
				logger.Error(err)
			}
		}
	}

	return nil
}
