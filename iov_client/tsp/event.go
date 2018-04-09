package tsp

import (
	"errors"
	"hcxy/iov/eventmsg"
	"hcxy/iov/log/logger"
	"hcxy/iov/message"
	"net"
)

type TboxEvent struct {
	tboxEventMsg    *eventmsg.EventMsg
	Conn            net.Conn
	EventMsgChannel chan *eventmsg.EventMsgChan
	Tid             uint32
	SendToMqChan    chan []byte
	RecvToMqChan    chan []byte
}

func NewTboxEvent(conn net.Conn, eventMsgChannel chan *eventmsg.EventMsgChan, tid uint32, sendToMqChan, recvToMqChan chan []byte) (*TboxEvent, error) {
	te := &TboxEvent{
		Conn:            conn,
		EventMsgChannel: eventMsgChannel,
		Tid:             tid,
		SendToMqChan:    sendToMqChan,
		RecvToMqChan:    recvToMqChan,
	}

	return te, nil
}

func (tb *TboxEvent) Start() error {
	tb.tboxEventMsg = &eventmsg.EventMsg{
		EventDispatcherCallback: tb.tboxEventDispatcher,
		EventMsgChannel:         tb.EventMsgChannel,
	}

	err := tb.tboxEventMsg.EventMsgScheduler()
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func (tb *TboxEvent) taskReadTcp() {
	for {
		msg := message.Message{
			Connection: tb.Conn,
			/*CallbackFn: tb.getAes128Key,*/
		}

		data, errorCode, err := msg.RecvOneMessage()
		if err != nil {
			logger.Error(err)
			if errorCode == message.ErrorCodeConnectionBreak {
				tb.tboxEventMsg.PushEventMsgChannel(EventConnectionClosed, nil) /*need go out*/
				return
			} else {
				continue
			}
		}

		tb.tboxEventMsg.PushEventMsgChannel(EventRecvedData, data)
	}
}

func (tb *TboxEvent) sendToMq(data []byte) error {
	newData, err := AddTidHeader(tb.Tid, data)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("Tsp recv data with tid =", newData)

	tb.SendToMqChan <- newData

	return nil
}

func (tb *TboxEvent) sendToTbox(data []byte) error {
	n, err := tb.Conn.Write(data[4:])
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("send data length =", n)
	return nil
}

func (tb *TboxEvent) tboxEventDispatcher(event int, param interface{}) error {
	logger.Debug("event =", GetEventName(event))

	switch event {
	case eventmsg.EventStart:
		go tb.taskReadTcp()

	case EventRecvedData:
		logger.Debug("Tsp recv data =", param.([]byte))
		tb.sendToMq(param.([]byte))

	case EventSendToTbox:
		tb.sendToTbox(param.([]byte))

	case EventConnectionClosed:
		return errors.New("Connection closed!")

	default:
		logger.Error("Unknown event!")
	}

	return nil
}
