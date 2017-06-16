package hello

import (
	context "golang.org/x/net/context"
	pb "github.com/cloudtrust/lbclient/examples/grpcHello/pb"
	"google.golang.org/grpc"
	"github.com/pkg/errors"
	"net"
	"fmt"
)

type greeter struct {
	s string
}

func (g *greeter)SayHi(context.Context, *pb.Empty) (*pb.HiMessage, error) {
	return &pb.HiMessage{g.s}, nil
}

func NewGreeter(name string) pb.GreeterServer {
	return &greeter{name}
}

func GetServer(port int, g pb.GreeterServer) (*grpc.Server, net.Listener, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to listen ")
	}
	grpcServer := grpc.NewServer()
	pb.RegisterGreeterServer(grpcServer, g)
	return grpcServer, lis, nil
}
