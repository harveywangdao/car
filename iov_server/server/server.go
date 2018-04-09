package server

import (
	//"hcxy/iov/iov_server/mqservice"
	//"hcxy/iov/iov_server/tboxservice"
	"hcxy/iov/iov_server/gateway"
	//"net"
	"sync"
)

func Server() {
	var wg sync.WaitGroup

	//tboxconnToTboxChan := make(chan net.Conn, 128)
	//mqToTboxChan := make(chan []byte, 128)
	//tboxToMqChan := make(chan []byte, 128)

	wg.Add(1)

	//go tboxservice.TBox(tboxconnToTboxChan, mqToTboxChan, tboxToMqChan)
	//go tboxservice.TBoxConn(tboxconnToTboxChan)

	gate := gateway.Gateway{}
	go gate.GatewayStart()

	//go mqservice.MQ(mqToTboxChan, tboxToMqChan)

	wg.Wait()
}
