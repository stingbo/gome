package main

import (
	"context"
	"gome/api"
	"log"
	"net"
	"strings"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	HOST string = "gome"
	PORT string = "8088"
)

type FormatData struct{}

func (fd *FormatData) DoFormat(ctx context.Context, in *api.Data) (out *api.Data, err error) {
	str := in.Text
	out = &api.Data{Text: strings.ToUpper(str)}
	return out, nil
}

func main() {
	listener, err := net.Listen("tcp", HOST+":"+PORT)
	if err != nil {
		log.Fatalln("faile listen at: " + HOST + ":" + PORT)
	} else {
		log.Println("Demo server is listening at: " + HOST + ":" + PORT)
	}

	rpcServer := grpc.NewServer()
	api.RegisterFormatDataServer(rpcServer, &FormatData{})
	reflection.Register(rpcServer)
	if err = rpcServer.Serve(listener); err != nil {
		log.Fatalln("faile serve at: " + HOST + ":" + PORT)
	}
}
