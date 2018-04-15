package gateway

import (
	"github.com/harveywangdao/road/log/logger"
	"net"
	"sync"
)

const (
	Port = ":6023"
)

type ThingConn struct {
	ThingID      string
	ThingService *Thing
}

type Gateway struct {
	ThingConns map[string]*Thing
	ThingChan  chan ThingConn
	lock       sync.Mutex
}

func (gate *Gateway) ShowAllThings() {
	for k, v := range gate.ThingConns {
		logger.Info("ThingID =", k, "Thing =", v)
	}
}

func (gate *Gateway) recvThingConnection() {
	for {
		select {
		case thingConn := <-gate.ThingChan:
			logger.Debug("thingConn.ThingID =", thingConn.ThingID)
			gate.lock.Lock()
			gate.ThingConns[thingConn.ThingID] = thingConn.ThingService
			gate.lock.Unlock()
			gate.ShowAllThings()
		}
	}
}

func (gate *Gateway) thingHandler(conn net.Conn) {
	defer conn.Close()
	logger.Debug("Net =", conn.LocalAddr().Network(), ", Addr =", conn.LocalAddr().String())
	logger.Debug("Remote net =", conn.RemoteAddr().Network(), ", Remote addr =", conn.RemoteAddr().String())

	msgChan := make(chan ThingMessage, 128)

	thing, err := NewThing(msgChan, conn, gate.ThingChan)
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
	gw.ThingChan = make(chan ThingConn, 128)
	gw.ThingConns = make(map[string]*Thing)

	go gw.recvThingConnection()

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
		logger.Info("Waiting connecting...")
		conn, err := listener.Accept()
		if err != nil {
			logger.Error(err)
			return
		}

		go gw.thingHandler(conn)
	}
}
