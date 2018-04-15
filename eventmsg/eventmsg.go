package eventmsg

import (
	"github.com/harveywangdao/road/log/logger"
	"time"
)

const (
	EventStart = iota
)

type EventMsg struct {
	EventDispatcherCallback EventDispatcherFn
	EventMsgChannel         chan *EventMsgChan
}

type EventMsgChan struct {
	Event int
	Param interface{}
}

type EventDispatcherFn func(event int, param interface{}) error

func NewEventMsg(fn EventDispatcherFn, ch chan *EventMsgChan) (*EventMsg, error) {
	em := &EventMsg{
		EventDispatcherCallback: fn,
		EventMsgChannel:         ch,
	}

	return em, nil
}

func (em *EventMsg) Close() error {
	return nil
}

func (em *EventMsg) PushEventMsgChannel(event int, param interface{}) {
	emc := &EventMsgChan{
		Event: event,
		Param: param,
	}

	em.EventMsgChannel <- emc
}

func (em *EventMsg) EventMsgScheduler() error {
	defer em.Close()

	em.PushEventMsgChannel(EventStart, nil)

	for {
		select {
		case eveMsg := <-em.EventMsgChannel:
			logger.Debug("eveMsg =", eveMsg)
			err := em.EventDispatcherCallback(eveMsg.Event, eveMsg.Param)
			if err != nil {
				logger.Error(err)
				return err
			}

		case <-time.After(time.Second * 60):
			logger.Info("Timeout!")
		}
	}

	return nil
}
