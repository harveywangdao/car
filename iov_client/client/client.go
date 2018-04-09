package client

import (
	"hcxy/iov/iov_client/tbox"
	//"hcxy/iov/iov_client/tsp"
	"sync"
)

func Client() {
	var wg sync.WaitGroup

	wg.Add(1)

	go tbox.TBox()
	//go tsp.Tsp()

	wg.Wait()
}
