package server

import (
	"github.com/harveywangdao/road/iot/gateway"
	"sync"
)

func Server() {
	var wg sync.WaitGroup

	wg.Add(1)

	gw := gateway.Gateway{}
	go gw.GatewayStart()

	wg.Wait()
}
