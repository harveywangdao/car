package main

import (
	"github.com/harveywangdao/road/log/logger"
	pb "github.com/harveywangdao/road/protobuf/adderserver"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

const (
	port = ":50051"
)

type server struct{}

func (s *server) Add(ctx context.Context, in *pb.AddRequest) (*pb.AddReply, error) {
	logger.Info("Addparameter1, Addparameter2 =", in.Addparameter1, in.Addparameter2)
	return &pb.AddReply{Sum: in.Addparameter1 + in.Addparameter2}, nil
}

func main() {
	logger.Info("Start adderserver...")

	lis, err := net.Listen("tcp", port)
	if err != nil {
		logger.Error("failed to listen:", err)
		return
	}

	s := grpc.NewServer()
	pb.RegisterAdderServer(s, &server{})

	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve:", err)
		return
	}
}
