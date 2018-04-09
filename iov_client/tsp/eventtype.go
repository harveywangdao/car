package tsp

import (
	"hcxy/iov/eventmsg"
	/*"hcxy/iov/log/logger"*/)

const (
	EventRecvedData = eventmsg.EventStart + 1 + iota
	EventSendToTbox
	EventConnectionClosed

	UnknownEventMessage
)

var (
	EventNameList = []string{
		"EventStart",
		"EventRecvedData",
		"EventSendToTbox",
		"EventConnectionClosed",
		"UnknownEventMessage",
	}
)

func GetEventName(event int) string {
	if event >= UnknownEventMessage {
		return "UnknownEventMessage"
	}

	return EventNameList[event]
}
