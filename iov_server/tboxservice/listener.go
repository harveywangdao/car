package tboxservice

import (
	"hcxy/iov/log/logger"
	"net"
)

func tboxServiceHandler(conn net.Conn) {
	defer conn.Close()
	logger.Debug("Net =", conn.LocalAddr().Network(), ", Addr =", conn.LocalAddr().String())
	logger.Debug("Remote net =", conn.RemoteAddr().Network(), ", Remote addr =", conn.RemoteAddr().String())

	eventMsgChan := make(chan EventMessage, 128)

	eve, err := NewEvent(eventMsgChan, 0, conn, true)
	if err != nil {
		logger.Error(err)
		return
	}

	err = eve.EventScheduler()
	if err != nil {
		logger.Error(err)
		return
	}
}

func ConnectListener(tboxconnToTboxChan chan net.Conn) {
	for {
		select {
		case conn := <-tboxconnToTboxChan:
			go tboxServiceHandler(conn)
		}
	}
}
