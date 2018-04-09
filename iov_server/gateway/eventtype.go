package gateway

import (
	"hcxy/iov/log/logger"
)

const (
	EventConnectionClosed = iota

	RegisterReqEventMessage
	RegisterAckEventMessage

	EventLoginRequest
	EventLoginChallenge
	EventLoginResponse
	EventLoginFailure
	EventLoginSuccess

	EventReLoginRequest
	EventReLoginAck

	EventHeartbeatRequest
	EventHeartbeatAck

	EventReadConfigRequest
	EventReadConfigAck

	EventRemoteOperationRequest
	EventDispatcherAckMessage
	EventRemoteOperationEnd
	EventRemoteOperationAck

	EventVHSUpdateMessage
	EventVHSUpdateMessageAck

	UnknownEventMessage
)

var (
	EventNameList = []string{
		"EventConnectionClosed",

		"RegisterReqEventMessage",
		"RegisterAckEventMessage",

		"EventLoginRequest",
		"EventLoginChallenge",
		"EventLoginResponse",
		"EventLoginFailure",
		"EventLoginSuccess",

		"EventReLoginRequest",
		"EventReLoginAck",

		"EventHeartbeatRequest",
		"EventHeartbeatAck",

		"EventReadConfigRequest",
		"EventReadConfigAck",

		"EventRemoteOperationRequest",
		"EventDispatcherAckMessage",
		"EventRemoteOperationEnd",
		"EventRemoteOperationAck",

		"EventVHSUpdateMessage",
		"EventVHSUpdateMessageAck",

		"UnknownEventMessage",
	}
)

type EventTypeMapAidMid struct {
	EventType int
	Aid       uint8
	Mid       uint8
}

var (
	eventTypeMapAidMidTable = [...]EventTypeMapAidMid{
		{RegisterReqEventMessage, 0x1, 0x1},
		{RegisterAckEventMessage, 0x1, 0x2},

		{EventLoginRequest, 0x2, 0x1},
		{EventLoginChallenge, 0x2, 0x2},
		{EventLoginResponse, 0x2, 0x3},
		{EventLoginFailure, 0x2, 0x4},
		{EventLoginSuccess, 0x2, 0x5},

		{EventReLoginRequest, 0x4, 0x1},
		{EventReLoginAck, 0x4, 0x2},

		{EventReadConfigRequest, 0x5, 0x1},
		{EventReadConfigAck, 0x5, 0x2},

		{EventHeartbeatRequest, 0xB, 0x1},
		{EventHeartbeatAck, 0xB, 0x2},

		{EventRemoteOperationRequest, 0xF1, 0x1},
		{EventDispatcherAckMessage, 0xF1, 0x2},
		{EventRemoteOperationEnd, 0xF1, 0x3},
		{EventRemoteOperationAck, 0xF1, 0x4},

		{EventVHSUpdateMessage, 0xF5, 0x4},
		{EventVHSUpdateMessageAck, 0xF5, 0x2},
	}
)

func GetEventTypeByAidMid(aid uint8, mid uint8) int {
	event := UnknownEventMessage

	for _, e := range eventTypeMapAidMidTable {
		if e.Aid == aid && e.Mid == mid {
			event = e.EventType
			break
		}
	}

	logger.Debug("event =", event)

	return event
}

func GetEventName(event int) string {
	if event >= UnknownEventMessage {
		return "UnknownEventMessage"
	}

	return EventNameList[event]
}

/*var (
	eventTypeTable = [][]int{
		{RegisterReqEventMessage, RegisterAckEventMessage},                                                 //0x1
		{EventLoginRequest, EventLoginChallenge, EventLoginResponse, EventLoginFailure, EventLoginSuccess}, //0x2
		{}, //0x3
		{EventReLoginRequest, EventReLoginAck}, //0x4
	}
)

func GetEventTypeByAidMid2(aid uint8, mid uint8) int {
	//logger.Info("eventTypeTable =", eventTypeTable)

	if aid > 0 && mid > 0 {
		if int(aid) <= len(eventTypeTable) {
			if int(mid) <= len(eventTypeTable[aid-1]) {
				return eventTypeTable[aid-1][mid-1]
			}
		}
	}

	return UnknownEventMessage
}*/
