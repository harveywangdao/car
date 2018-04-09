package tboxservice

import (
	"hcxy/iov/log/logger"
	"net"
)

const (
	Port = ":1235"
)

func TBoxConn(tboxconnToTboxChan chan net.Conn) {
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
		tboxconnToTboxChan <- conn
	}
}
