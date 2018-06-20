package gateway

import (
	"encoding/json"
	"github.com/harveywangdao/road/log/logger"
	"net"
	"sync"
)

const (
	Port = ":6024"
)

type ThingConn struct {
	ThingID      string
	ThingService *Thing
}

type Gateway struct {
	ThingConns      map[string]*Thing
	AddThingChan    chan ThingConn
	DeleteThingChan chan ThingConn
	lock            sync.Mutex
}

func (gw *Gateway) ShowAllThings() {
	for k, v := range gw.ThingConns {
		logger.Debug("ThingID =", k, "Thing =", v)
	}
}

func (gw *Gateway) recvThingConnection() {
	for {
		select {
		case addThingConn := <-gw.AddThingChan:
			logger.Info("addThingConn.ThingID =", addThingConn.ThingID)
			gw.lock.Lock()
			gw.ThingConns[addThingConn.ThingID] = addThingConn.ThingService
			gw.lock.Unlock()
			gw.ShowAllThings()

		case deleteThingConn := <-gw.DeleteThingChan:
			logger.Info("deleteThingConn.ThingID =", deleteThingConn.ThingID)
			gw.lock.Lock()
			delete(gw.ThingConns, deleteThingConn.ThingID)
			gw.lock.Unlock()
			gw.ShowAllThings()
		}
	}
}

type WebRequest struct {
	ThingId string `json:"thingid"`
	Command string `json:"command"`
}

type WebResponse struct {
	ThingId string `json:"thingid"`
	Status  string `json:"status"`
}

func (gw *Gateway) WebTask() {
	WebToGatewayChan := make(chan []byte, 128)
	gatewayToWebChan := make(chan []byte, 128)

	go func() {
		for {
			select {
			case data := <-WebToGatewayChan:
				logger.Info("data from web =", string(data))

				webRequest := &WebRequest{}
				err := json.Unmarshal(data, webRequest)
				if err != nil {
					logger.Error(err)
					return
				}

				//{"thingid":"fsdvsdvsdvsdv","command":"lock"}
				//{"thingid":"WDDUX52684DFR4582","command":"lock"}
				var webResponse *WebResponse

				gw.lock.Lock()
				thing, ok := gw.ThingConns[webRequest.ThingId]
				if ok {
					thing.PushEventChannel2(EventRemoteOperationRequest, nil, webRequest.Command)

					webResponse = &WebResponse{
						ThingId: webRequest.ThingId,
						Status:  "success",
					}
				} else {
					logger.Error("Can not find the thing, thingid = ", webRequest.ThingId)
					webResponse = &WebResponse{
						ThingId: webRequest.ThingId,
						Status:  "fail",
					}
				}
				gw.lock.Unlock()

				webResponseJson, err := json.Marshal(webResponse)
				if err != nil {
					logger.Error(err)
					return
				}

				gatewayToWebChan <- webResponseJson
			}
		}
	}()

	listenMQ := ListenMQ{
		ConsumerRoutineNum: 1,
		ProducerRoutineNum: 10,
		MQAddr:             "localhost:9092",
		RecvMessageTopic:   "WebToGateway",
		SendMessageTopic:   "GatewayToWeb",
		MqToGatewayChan:    WebToGatewayChan,
		GatewayToMqChan:    gatewayToWebChan,
	}

	listenMQ.Run()
}

func (gw *Gateway) thingHandler(conn net.Conn) {
	defer conn.Close()
	logger.Debug("Net =", conn.LocalAddr().Network(), ", Addr =", conn.LocalAddr().String())
	logger.Debug("Remote net =", conn.RemoteAddr().Network(), ", Remote addr =", conn.RemoteAddr().String())

	msgChan := make(chan ThingMessage, 128)

	thing, err := NewThing(msgChan, conn, gw.AddThingChan, gw.DeleteThingChan)
	if err != nil {
		logger.Error(err)
		return
	}

	err = thing.ThingScheduler()
	if err != nil {
		logger.Error(err)
		return
	}
}

func (gw *Gateway) GatewayStart() {
	gw.AddThingChan = make(chan ThingConn, 128)
	gw.DeleteThingChan = make(chan ThingConn, 128)
	gw.ThingConns = make(map[string]*Thing)

	go gw.recvThingConnection()
	go gw.WebTask()

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

	logger.Info("Net =", listener.Addr().Network(), ", Addr =", listener.Addr().String())

	for {
		logger.Debug("Waiting connecting...")
		conn, err := listener.Accept()
		if err != nil {
			logger.Error(err)
			return
		}

		go gw.thingHandler(conn)
	}
}
