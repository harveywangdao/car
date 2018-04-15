package client

import (
	"github.com/harveywangdao/road/iot_client/thing"
	"sync"
)

func Client() {
	var wg sync.WaitGroup

	wg.Add(1)

	go thing.Things()

	wg.Wait()
}
