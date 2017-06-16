package hello

import (
	"testing"
	"google.golang.org/grpc"
	"github.com/stretchr/testify/require"
	pb "github.com/cloudtrust/lbclient/examples/grpcHello/pb"
	context "golang.org/x/net/context"
	"fmt"
	"net"
	"github.com/stretchr/testify/assert"
	lbgrpc "github.com/cloudtrust/lbclient/grpc"
)


func TestBal(tester *testing.T) {
	var respForm string = "Hello %d"
	var port int = 33333
	var num int = 5
	var numTest int = 10
	var ech chan error = make(chan error, num)
	var err error
	var addrs []string
	for i:=0;i<num;i++ {
		var serv *grpc.Server
		var lis net.Listener
		var conn string = fmt.Sprintf("127.0.0.1:%d", port+i)
		{
			lis, err = net.Listen("tcp", conn)
			require.Nil(tester, err, "Error while trying to listen!", err)
			require.NotNil(tester, lis, "Listener is nil!!")
			addrs = append(addrs, conn)
			serv = grpc.NewServer()
			pb.RegisterGreeterServer(serv, NewGreeter(fmt.Sprintf(respForm, port+i)))
			go func(s *grpc.Server, l net.Listener) {
				ech <- s.Serve(l)
			}(serv, lis)
			defer func(s *grpc.Server) {
				s.Stop()
				fmt.Println(<-ech)
			}(serv)
		}
	}
	var bal grpc.Balancer
	var conn *grpc.ClientConn
	{
		bal = grpc.RoundRobin(lbgrpc.NewStaticResolver(addrs))
		conn, err = grpc.Dial("hello", grpc.WithBlock(),
			grpc.WithInsecure(), grpc.WithBalancer(bal))
		require.Nil(tester, err, "Didn't work", err)
	}
	defer conn.Close()
	var client pb.GreeterClient = pb.NewGreeterClient(conn)
	for i:=0; i<numTest; i++ {
		res, err := client.SayHi(context.Background(), &pb.Empty{})
		assert.Nil(tester, err, "Error not nil")
		fmt.Println(res.GetValue())
	}
}



