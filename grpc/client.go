package grpc

import (
	"google.golang.org/grpc"
	"log"
	"net"
)

func NewRpcClient() *grpc.ClientConn {
	host := Conf.GRPCConf.Host
	port := Conf.GRPCConf.Port

	conn, err := grpc.Dial(net.JoinHostPort(host, port), grpc.WithInsecure())
	if err != nil {
		log.Fatalln("Can't connect: " + host + ":" + port)
	}

	return conn
}
