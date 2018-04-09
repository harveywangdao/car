package gateway

import (
	"encoding/json"
	"hcxy/iov/log/logger"
	"net"
	"sync"
)

const (
	Port = ":1235"
)

type TboxConn struct {
	VIN         string
	TboxService *Tbox
}

type Gateway struct {
	TboxConns map[string]*Tbox
	TboxChan  chan TboxConn
	lock      sync.Mutex
}

func (gate *Gateway) ShowAllTboxes() {
	for k, v := range gate.TboxConns {
		logger.Info("vin =", k, "tbox =", v)
	}
}

func (gate *Gateway) RecvTboxConnection() {
	for {
		select {
		case tboxConn := <-gate.TboxChan:
			logger.Debug("tboxConn.VIN =", tboxConn.VIN)
			gate.lock.Lock()
			gate.TboxConns[tboxConn.VIN] = tboxConn.TboxService
			gate.lock.Unlock()
			gate.ShowAllTboxes()
		}
	}
}

func (gate *Gateway) tboxHandler(conn net.Conn) {
	defer conn.Close()
	logger.Debug("Net =", conn.LocalAddr().Network(), ", Addr =", conn.LocalAddr().String())
	logger.Debug("Remote net =", conn.RemoteAddr().Network(), ", Remote addr =", conn.RemoteAddr().String())

	msgChan := make(chan TboxMessage, 128)

	tbox, err := NewTbox(msgChan, conn, gate.TboxChan)
	if err != nil {
		logger.Error(err)
		return
	}

	err = tbox.TboxScheduler()
	if err != nil {
		logger.Error(err)
		return
	}
}

type FinanceRequest struct {
	Vin     string `json:"vin"`
	Command string `json:"command"`
}

type FinanceResponse struct {
	Vin    string `json:"vin"`
	Status string `json:"status"`
}

func (gate *Gateway) FinanceLeaseTask() {
	financeToGatewayChan := make(chan []byte, 128)
	gatewayToFinanceChan := make(chan []byte, 128)

	go func() {
		for {
			select {
			case data := <-financeToGatewayChan:
				logger.Info("data from finance =", data)

				financeRequest := &FinanceRequest{}
				err := json.Unmarshal(data, financeRequest)
				if err != nil {
					logger.Error(err)
					return
				}
				//{"vin":"fsdvsdvsdvsdv","command":"lock"}
				//{"vin":"WDDUX52684DFR4582","command":"lock"}
				var financeResponse *FinanceResponse

				gate.lock.Lock()
				tbox, ok := gate.TboxConns[financeRequest.Vin]
				if ok {
					tbox.PushEventChannel2(EventRemoteOperationRequest, nil, financeRequest.Command)

					financeResponse = &FinanceResponse{
						Vin:    financeRequest.Vin,
						Status: "success",
					}
				} else {
					logger.Error("Can not find the tbox, vin = ", financeRequest.Vin)
					financeResponse = &FinanceResponse{
						Vin:    financeRequest.Vin,
						Status: "fail",
					}
				}
				gate.lock.Unlock()

				financeResponseJson, err := json.Marshal(financeResponse)
				if err != nil {
					logger.Error(err)
					return
				}

				gatewayToFinanceChan <- financeResponseJson
			}
		}
	}()

	financeLease := FinanceLease{
		ConsumerRoutineNum: 1,
		ProducerRoutineNum: 10,
		MQAddr:             "localhost:9092",
		RecvMessageTopic:   "FinanceLeaseLockCarToGateway",
		SendMessageTopic:   "GatewayToFinanceLeaseLockCar",
		MqToGatewayChan:    financeToGatewayChan,
		GatewayToMqChan:    gatewayToFinanceChan,
	}

	financeLease.FinanceLeaseStart()
}

func (gate *Gateway) GatewayStart() {
	gate.TboxChan = make(chan TboxConn, 128)
	gate.TboxConns = make(map[string]*Tbox)

	go gate.RecvTboxConnection()
	go gate.FinanceLeaseTask()

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

		go gate.tboxHandler(conn)
	}
}
